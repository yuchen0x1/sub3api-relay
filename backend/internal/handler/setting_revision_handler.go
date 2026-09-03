package handler

import (
	"strconv"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingRevisionHandler 管理台配置版本历史接口。
type SettingRevisionHandler struct {
	settingService *service.SettingService
}

// NewSettingRevisionHandler 创建配置版本 handler。
func NewSettingRevisionHandler(settingService *service.SettingService) *SettingRevisionHandler {
	return &SettingRevisionHandler{settingService: settingService}
}

// List GET /api/v1/admin/setting-revisions
func (h *SettingRevisionHandler) List(c *gin.Context) {
	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	items, total, err := h.settingService.ListSettingRevisions(c.Request.Context(), limit, offset)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// Rollback POST /api/v1/admin/setting-revisions/:id/rollback
func (h *SettingRevisionHandler) Rollback(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid id")
		return
	}
	if err := h.settingService.RollbackSettingRevision(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"rolled_back": id})
}
