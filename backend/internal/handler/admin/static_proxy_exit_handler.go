package admin

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/pagination"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// StaticProxyExitHandler handles admin management of account-owned static proxy
// exit assignments.
type StaticProxyExitHandler struct {
	svc *service.StaticProxyExitService
}

// NewStaticProxyExitHandler creates a new static proxy exit handler.
func NewStaticProxyExitHandler(svc *service.StaticProxyExitService) *StaticProxyExitHandler {
	return &StaticProxyExitHandler{svc: svc}
}

// Service exposes the backing static proxy exit service so the admin account
// handler can reuse the same instance for capacity enrichment without taking a
// second constructor dependency.
func (h *StaticProxyExitHandler) Service() *service.StaticProxyExitService {
	if h == nil {
		return nil
	}
	return h.svc
}

// List handles listing static proxy exits with pagination.
// GET /api/v1/admin/static-proxy-exits
func (h *StaticProxyExitHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)

	params := service.StaticProxyExitListParams{
		PaginationParams: pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    c.DefaultQuery("sort_by", "id"),
			SortOrder: c.DefaultQuery("sort_order", "desc"),
		},
		AccountID:    parseOptionalInt64Query(c, "account_id"),
		Enabled:      parseOptionalBoolQuery(c, "enabled"),
		HealthStatus: strings.TrimSpace(c.Query("health_status")),
		Search:       strings.TrimSpace(c.Query("search")),
	}

	items, total, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, staticProxyExitsFromService(items), total, page, pageSize)
}

// GetByID handles getting a static proxy exit by ID.
// GET /api/v1/admin/static-proxy-exits/:id
func (h *StaticProxyExitHandler) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	exit, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, staticProxyExitFromService(exit))
}

// Create handles creating a new static proxy exit.
// POST /api/v1/admin/static-proxy-exits
func (h *StaticProxyExitHandler) Create(c *gin.Context) {
	var req StaticProxyExitCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	exit, err := h.svc.Create(c.Request.Context(), service.StaticProxyExitCreateInput{
		AccountID: req.AccountID,
		ProxyID:   req.ProxyID,
		Name:      req.Name,
		Position:  req.Position,
		Enabled:   req.Enabled,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, staticProxyExitFromService(exit))
}

// Update handles updating a static proxy exit.
// PUT /api/v1/admin/static-proxy-exits/:id
func (h *StaticProxyExitHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req StaticProxyExitUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	exit, err := h.svc.Update(c.Request.Context(), id, service.StaticProxyExitUpdateInput{
		AccountID: req.AccountID,
		ProxyID:   req.ProxyID,
		Name:      req.Name,
		Position:  req.Position,
		Enabled:   req.Enabled,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, staticProxyExitFromService(exit))
}

// Toggle handles enabling/disabling a static proxy exit.
// POST /api/v1/admin/static-proxy-exits/:id/toggle
func (h *StaticProxyExitHandler) Toggle(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req StaticProxyExitToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	exit, err := h.svc.Toggle(c.Request.Context(), id, req.Enabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, staticProxyExitFromService(exit))
}

// Delete handles deleting a static proxy exit.
// DELETE /api/v1/admin/static-proxy-exits/:id
func (h *StaticProxyExitHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Static proxy exit deleted successfully"})
}

// GetCapacitySummary handles fetching the capacity summary (optionally scoped
// to an account via the account_id query parameter).
// GET /api/v1/admin/static-proxy-exits/capacity
func (h *StaticProxyExitHandler) GetCapacitySummary(c *gin.Context) {
	accountID := parseOptionalInt64Query(c, "account_id")
	summary, err := h.svc.GetCapacitySummary(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, staticProxyExitCapacitySummaryFromService(summary))
}

// ListByAccount handles listing static proxy exits for a specific account.
// GET /api/v1/admin/static-proxy-exits/account/:account_id
func (h *StaticProxyExitHandler) ListByAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	page, pageSize := response.ParsePagination(c)

	params := service.StaticProxyExitListParams{
		PaginationParams: pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    c.DefaultQuery("sort_by", "id"),
			SortOrder: c.DefaultQuery("sort_order", "desc"),
		},
		AccountID:    &accountID,
		Enabled:      parseOptionalBoolQuery(c, "enabled"),
		HealthStatus: strings.TrimSpace(c.Query("health_status")),
		Search:       strings.TrimSpace(c.Query("search")),
	}

	items, total, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, staticProxyExitsFromService(items), total, page, pageSize)
}

// GetAccountCapacity handles fetching the capacity summary for a single
// account via the nested account route.
// GET /api/v1/admin/static-proxy-exits/account/:account_id/capacity
func (h *StaticProxyExitHandler) GetAccountCapacity(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	summary, err := h.svc.GetCapacitySummary(c.Request.Context(), &accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, staticProxyExitCapacitySummaryFromService(summary))
}

func parseOptionalInt64Query(c *gin.Context, key string) *int64 {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	return &v
}

func parseOptionalBoolQuery(c *gin.Context, key string) *bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}
