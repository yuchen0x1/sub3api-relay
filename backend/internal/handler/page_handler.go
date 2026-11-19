package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	middleware2 "github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

var validSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

const maxPageFileSize = 1 << 20 // 1MB

type PageHandler struct {
	pagesDir       string
	settingService *service.SettingService
}

func NewPageHandler(dataDir string, settingService *service.SettingService) *PageHandler {
	pagesDir := filepath.Join(dataDir, "pages")
	_ = os.MkdirAll(pagesDir, 0755)
	return &PageHandler{pagesDir: pagesDir, settingService: settingService}
}

// GetPageContent serves raw markdown content for a given slug.
// GET /api/v1/pages/:slug
func (h *PageHandler) GetPageContent(c *gin.Context) {
	slug := c.Param("slug")
	if !validSlugPattern.MatchString(slug) || len(slug) > 64 {
		response.BadRequest(c, "Invalid page slug")
		return
	}

	// Visibility check: slug must be configured in custom_menu_items
	// and the user must have permission based on visibility setting
	if !h.checkSlugVisibility(c, slug) {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}

	root, err := os.OpenRoot(h.pagesDir)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	defer func() { _ = root.Close() }()

	file, info, err := openRegularPageFile(root, slug+".md")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "page not found"})
		return
	}
	defer func() { _ = file.Close() }()
	if info.Size() > maxPageFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "page too large"})
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, maxPageFileSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read page"})
		return
	}
	if len(content) > maxPageFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "page too large"})
		return
	}

	c.Data(http.StatusOK, "text/markdown; charset=utf-8", content)
}

// ListPages returns available page slugs.
// GET /api/v1/pages
func (h *PageHandler) ListPages(c *gin.Context) {
	entries, err := os.ReadDir(h.pagesDir)
	if err != nil {
		response.Success(c, []string{})
		return
	}

	slugs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") {
			slugs = append(slugs, strings.TrimSuffix(name, ".md"))
		}
	}
	response.Success(c, slugs)
}

// ServePageImage serves images from data/pages/{slug}/ directory.
// GET /api/v1/pages/:slug/images/*filename
// No JWT required (browser img tags can't carry tokens), but visibility is checked.
func (h *PageHandler) ServePageImage(c *gin.Context) {
	slug := c.Param("slug")
	filename := c.Param("filename")
	filename = strings.TrimPrefix(filename, "/")

	if !validSlugPattern.MatchString(slug) || len(slug) > 64 {
		c.Status(http.StatusNotFound)
		return
	}

	if !h.checkImageSlugVisibility(c, slug) {
		c.Status(http.StatusNotFound)
		return
	}

	imagesDir, err := openPageImagesDir(h.pagesDir, slug)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer func() { _ = imagesDir.Close() }()

	relPath, ok := cleanPageImageRelativePath(filename)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	file, info, err := openRegularPageFile(imagesDir, relPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()
	http.ServeContent(c.Writer, c.Request, filepath.Base(relPath), info.ModTime(), file)
}

func openPageImagesDir(pagesDir, slug string) (*os.Root, error) {
	root, err := os.OpenRoot(pagesDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.OpenRoot(slug)
}

// Check the same handle that will be used to read or serve the file.
func openRegularPageFile(root *os.Root, name string) (*os.File, os.FileInfo, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		if err == nil {
			err = os.ErrNotExist
		}
		return nil, nil, err
	}
	return file, info, nil
}

func cleanPageImageRelativePath(filename string) (string, bool) {
	if filename == "" {
		return "", false
	}
	if strings.HasPrefix(filename, "/") {
		return "", false
	}
	decoded, err := url.PathUnescape(filename)
	if err != nil {
		return "", false
	}
	if decoded == "" || strings.HasPrefix(decoded, "/") || strings.Contains(decoded, "\\") || strings.ContainsRune(decoded, 0) {
		return "", false
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(decoded, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", false
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", false
	}

	relPath := filepath.Join(parts...)
	if filepath.IsAbs(relPath) || filepath.VolumeName(relPath) != "" {
		return "", false
	}
	return relPath, true
}

// findSlugVisibility looks up the slug in custom_menu_items and returns (visibility, found).
func (h *PageHandler) findSlugVisibility(c *gin.Context, slug string) (string, bool) {
	if h.settingService == nil {
		return "", false
	}

	raw := h.settingService.GetCustomMenuItemsRaw(c.Request.Context())
	if raw == "" || raw == "[]" {
		return "", false
	}

	var items []struct {
		URL        string `json:"url"`
		PageSlug   string `json:"page_slug"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return "", false
	}

	for _, item := range items {
		itemSlug := item.PageSlug
		if itemSlug == "" && strings.HasPrefix(item.URL, "md:") {
			itemSlug = strings.TrimPrefix(item.URL, "md:")
		}
		if itemSlug == slug {
			return item.Visibility, true
		}
	}
	return "", false
}

// checkSlugVisibility verifies the slug is configured in custom_menu_items
// and the authenticated user has permission to view it.
func (h *PageHandler) checkSlugVisibility(c *gin.Context, slug string) bool {
	visibility, found := h.findSlugVisibility(c, slug)
	if !found {
		return false
	}
	if visibility == "admin" {
		role, _ := middleware2.GetUserRoleFromContext(c)
		return role == "admin"
	}
	return true
}

// checkImageSlugVisibility checks visibility for image requests (no JWT available).
// Only allows user-visible pages; admin-only pages are blocked.
func (h *PageHandler) checkImageSlugVisibility(c *gin.Context, slug string) bool {
	visibility, found := h.findSlugVisibility(c, slug)
	if !found {
		return false
	}
	return visibility != "admin"
}

// RegisterPageRoutes registers page routes on a router group.
func RegisterPageRoutes(v1 *gin.RouterGroup, dataDir string, jwtAuth gin.HandlerFunc, adminAuth gin.HandlerFunc, settingService *service.SettingService) {
	h := NewPageHandler(dataDir, settingService)

	// Authenticated page content (JWT required + visibility check)
	pages := v1.Group("/pages")
	pages.Use(jwtAuth)
	{
		pages.GET("/:slug", h.GetPageContent)
	}

	// Images: no JWT (browser img tags can't carry tokens), visibility check in handler
	pageImages := v1.Group("/pages")
	{
		pageImages.GET("/:slug/images/*filename", h.ServePageImage)
	}

	// Admin-only: list all available pages
	adminPages := v1.Group("/pages")
	adminPages.Use(adminAuth)
	adminPages.Use(middleware2.AdminComplianceGuard(settingService))
	{
		adminPages.GET("", h.ListPages)
	}
}
