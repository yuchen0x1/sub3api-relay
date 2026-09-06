package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/httpclient"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type githubReleaseClient struct {
	httpClient         *http.Client
	downloadHTTPClient *http.Client
	updateGitHubToken  string
	updateGiteeToken   string
}

type githubReleaseClientError struct {
	err error
}

// NewGitHubReleaseClient 创建 GitHub Release 客户端
// proxyURL 为空时直连 GitHub，支持 http/https/socks5/socks5h 协议
// 代理配置失败时行为由 allowDirectOnProxyError 控制：
//   - false（默认）：返回错误占位客户端，禁止回退到直连
//   - true：回退到直连（仅限管理员显式开启）
func NewGitHubReleaseClient(proxyURL string, allowDirectOnProxyError bool) service.GitHubReleaseClient {
	// 安全说明：httpclient.GetClient 的错误链（url.Parse / proxyutil）不含明文代理凭据，
	// 但仍通过 slog 仅在服务端日志记录，不会暴露给 HTTP 响应。
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  30 * time.Second,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		sharedClient = &http.Client{Timeout: 30 * time.Second}
	}
	apiClient := cloneHTTPClient(sharedClient)
	apiClient.CheckRedirect = githubAPICheckRedirect(apiClient.CheckRedirect)

	// 下载客户端需要更长的超时时间
	downloadClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  10 * time.Minute,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy download client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		downloadClient = &http.Client{Timeout: 10 * time.Minute}
	}
	downloadClient = cloneHTTPClient(downloadClient)

	return &githubReleaseClient{
		httpClient:         apiClient,
		downloadHTTPClient: downloadClient,
		updateGitHubToken:  os.Getenv("UPDATE_GITHUB_TOKEN"),
		updateGiteeToken:   os.Getenv("UPDATE_GITEE_TOKEN"),
	}
}

func cloneHTTPClient(client *http.Client) *http.Client {
	cloned := *client
	return &cloned
}

func isGitHubAPIURL(url *url.URL) bool {
	return url != nil && strings.EqualFold(url.Scheme, "https") && url.User == nil &&
		strings.EqualFold(url.Host, "api.github.com")
}

func githubAPICheckRedirect(previous func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !isGitHubAPIURL(req.URL) {
			req.Header.Del("Authorization")
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
}

func (c *githubReleaseClient) newAPIRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Sub3API-Updater")
	if c.updateGitHubToken != "" && isGitHubAPIURL(req.URL) {
		req.Header.Set("Authorization", "Bearer "+c.updateGitHubToken)
	}
	return req, nil
}

func (c *githubReleaseClient) newGiteeAPIRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Gitee API 使用单独的 UPDATE_GITEE_TOKEN，避免把 GitHub 令牌发往 Gitee。
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Sub3API-Updater")
	if c.updateGiteeToken != "" {
		req.Header.Set("Authorization", "token "+c.updateGiteeToken)
	}
	return req, nil
}

func (c *githubReleaseClientError) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	return c.err
}

func (c *githubReleaseClientError) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	return nil, c.err
}

// FetchLatestRelease 获取最新 release。
// 项目自身自更新固定使用 Gitee；其他仓库（如 openai/codex 版本同步）
// 仍保持纯 GitHub，不受本项目发布通道影响。
func (c *githubReleaseClient) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	if repo != service.GitHubUpdateRepo {
		return c.fetchLatestReleaseFromGitHub(ctx, repo)
	}

	release, err := c.fetchLatestReleaseFromGitee(ctx)
	if err != nil {
		return nil, fmt.Errorf("Gitee update check failed: %w", err)
	}
	slog.Info("update_channel: latest project version fetched from gitee", "tag", release.TagName)
	return release, nil
}

func (c *githubReleaseClient) fetchLatestReleaseFromGitHub(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// fetchLatestReleaseFromGitee 走 Gitee API v5（响应结构与 GitHub 兼容：
// tag_name / assets[].browser_download_url，可直接复用 GitHubRelease 结构体）
func (c *githubReleaseClient) fetchLatestReleaseFromGitee(ctx context.Context) (*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/releases/latest", service.GiteeUpdateRepo)

	release, err := c.fetchGiteeURL(ctx, url)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(release.TagName) == "" {
		return nil, fmt.Errorf("Gitee release response missing tag_name")
	}

	return release, nil
}

// fetchGiteeURL 请求 Gitee API 并解析为 GitHubRelease。
// 配置的 UPDATE_GITEE_TOKEN 失效时 Gitee 对公开端点也返回 401/403——
// 此时必须去掉凭据匿名重试，而不是让一个过期 token 挡死公开的更新检查。
func (c *githubReleaseClient) fetchGiteeURL(ctx context.Context, url string) (*service.GitHubRelease, error) {
	fetch := func(sendAuth bool) (*http.Response, error) {
		req, err := c.newGiteeAPIRequest(ctx, url)
		if err != nil {
			return nil, err
		}
		if !sendAuth {
			req.Header.Del("Authorization")
		}
		return c.httpClient.Do(req)
	}

	resp, err := fetch(true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// Token 失效/无权限：匿名重试公开端点。
		resp.Body.Close()
		resp, err = fetch(false)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode == http.StatusNotFound {
		// A pushed tag is not necessarily a Gitee Release. Use the tag list for
		// version detection so source and Docker deployments can still see the
		// update; binary installation will continue to require release assets.
		return c.fetchLatestTagFromGitee(ctx)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gitee API returned %d", resp.StatusCode)
	}

	var release service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

type giteeTag struct {
	Name   string `json:"name"`
	Commit struct {
		CreatedAt string `json:"created_at"`
		Date      string `json:"date"`
	} `json:"commit"`
}

// fetchGiteeTags returns tags in the order supplied by Gitee (newest first).
// The endpoint is deliberately used only as a detection fallback: tags do not
// carry downloadable release assets.
func (c *githubReleaseClient) fetchGiteeTags(ctx context.Context, perPage int) ([]giteeTag, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	apiURL := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/tags?page=1&per_page=%d", service.GiteeUpdateRepo, perPage)
	req, err := c.newGiteeAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gitee tags API returned %d", resp.StatusCode)
	}
	var tags []giteeTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func strictGiteeTagVersion(name string) bool {
	name = strings.TrimPrefix(strings.TrimSpace(name), "v")
	parts := strings.Split(name, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func (c *githubReleaseClient) fetchLatestTagFromGitee(ctx context.Context) (*service.GitHubRelease, error) {
	tags, err := c.fetchGiteeTags(ctx, 100)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if !strictGiteeTagVersion(name) {
			continue
		}
		publishedAt := tag.Commit.CreatedAt
		if publishedAt == "" {
			publishedAt = tag.Commit.Date
		}
		return &service.GitHubRelease{
			TagName:     name,
			Name:        "Sub3API " + name,
			PublishedAt: publishedAt,
			HTMLURL:     fmt.Sprintf("https://gitee.com/%s/tree/%s", service.GiteeUpdateRepo, url.PathEscape(name)),
		}, nil
	}
	return nil, fmt.Errorf("Gitee tags response contains no semantic version tag")
}

func (c *githubReleaseClient) fetchReleasesFromGiteeTags(ctx context.Context, perPage int) ([]*service.GitHubRelease, error) {
	tags, err := c.fetchGiteeTags(ctx, perPage)
	if err != nil {
		return nil, err
	}
	releases := make([]*service.GitHubRelease, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if !strictGiteeTagVersion(name) {
			continue
		}
		publishedAt := tag.Commit.CreatedAt
		if publishedAt == "" {
			publishedAt = tag.Commit.Date
		}
		releases = append(releases, &service.GitHubRelease{
			TagName:     name,
			Name:        "Sub3API " + name,
			PublishedAt: publishedAt,
			HTMLURL:     fmt.Sprintf("https://gitee.com/%s/tree/%s", service.GiteeUpdateRepo, url.PathEscape(name)),
		})
	}
	return releases, nil
}

// FetchRecentReleases 获取最近的 release 列表（用于回滚版本列表）。
// 通道选择逻辑与 FetchLatestRelease 一致，仅作用于本项目自更新仓库。
func (c *githubReleaseClient) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100 // GitHub API hard limit
	}

	if repo != service.GitHubUpdateRepo {
		return c.fetchRecentReleasesFromGitHub(ctx, repo, perPage)
	}

	// Project rollback candidates are Gitee-only as well. A tag-only fallback
	// keeps the version list useful before release assets are attached.
	releases, err := c.fetchRecentReleasesFromGitee(ctx, perPage)
	if err != nil {
		return nil, fmt.Errorf("Gitee releases fetch failed: %w", err)
	}
	if len(releases) == 0 {
		return c.fetchReleasesFromGiteeTags(ctx, perPage)
	}
	return releases, nil
}

func (c *githubReleaseClient) fetchRecentReleasesFromGitHub(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", repo, perPage)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []*service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// fetchRecentReleasesFromGitee 走 Gitee API v5 列出最近 release（新→旧）
func (c *githubReleaseClient) fetchRecentReleasesFromGitee(ctx context.Context, perPage int) ([]*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/releases?page=1&per_page=%d&direction=desc", service.GiteeUpdateRepo, perPage)

	req, err := c.newGiteeAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return []*service.GitHubRelease{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gitee API returned %d", resp.StatusCode)
	}

	var releases []*service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (c *githubReleaseClient) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// 使用预配置的下载客户端（已包含代理配置）
	resp, err := c.downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// SECURITY: Check Content-Length if available
	if resp.ContentLength > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, maxSize)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	// SECURITY: Use LimitReader to enforce max download size even if Content-Length is missing/wrong
	limited := io.LimitReader(resp.Body, maxSize+1)
	written, err := io.Copy(out, limited)

	// Close file before attempting to remove (required on Windows)
	_ = out.Close()

	if err != nil {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return err
	}

	// Check if we hit the limit (downloaded more than maxSize)
	if written > maxSize {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return fmt.Errorf("download exceeded maximum size of %d bytes", maxSize)
	}

	return nil
}

func (c *githubReleaseClient) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
