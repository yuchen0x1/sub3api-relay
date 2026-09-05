package handler

import (
	"strconv"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	"github.com/gin-gonic/gin"
)

// GatewayBodyLogHandler 管理台请求体日志查看/重放接口。
type GatewayBodyLogHandler struct {
	service *service.GatewayBodyLogService
}

// NewGatewayBodyLogHandler 创建请求体日志 handler。
func NewGatewayBodyLogHandler(svc *service.GatewayBodyLogService) *GatewayBodyLogHandler {
	return &GatewayBodyLogHandler{service: svc}
}

// List GET /api/v1/admin/gateway-body-logs
func (h *GatewayBodyLogHandler) List(c *gin.Context) {
	if h.service == nil {
		response.BadRequest(c, "Gateway body log is not available")
		return
	}
	filter := service.GatewayBodyLogFilter{
		RequestID: c.Query("request_id"),
	}
	if v := c.Query("api_key_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.APIKeyID = id
		}
	}
	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.UserID = id
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	items, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 列表不带 body 正文，避免一次拉取大量文本；详情按 ID 获取。
	type item struct {
		ID        int64  `json:"id"`
		RequestID string `json:"request_id"`
		Method    string `json:"method"`
		Path      string `json:"path"`
		APIKeyID  int64  `json:"api_key_id"`
		UserID    int64  `json:"user_id"`
		Model     string `json:"model"`
		CreatedAt string `json:"created_at"`
	}
	out := make([]item, 0, len(items))
	for _, e := range items {
		out = append(out, item{
			ID:        e.ID,
			RequestID: e.RequestID,
			Method:    e.Method,
			Path:      e.Path,
			APIKeyID:  e.APIKeyID,
			UserID:    e.UserID,
			Model:     e.Model,
			CreatedAt: e.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	response.Success(c, gin.H{"items": out, "total": total})
}

// Get GET /api/v1/admin/gateway-body-logs/:id
func (h *GatewayBodyLogHandler) Get(c *gin.Context) {
	if h.service == nil {
		response.BadRequest(c, "Gateway body log is not available")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid id")
		return
	}
	entry, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry)
}

// Replay POST /api/v1/admin/gateway-body-logs/:id/replay
func (h *GatewayBodyLogHandler) Replay(c *gin.Context) {
	if h.service == nil {
		response.BadRequest(c, "Gateway body log is not available")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid id")
		return
	}
	result, err := h.service.Replay(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
