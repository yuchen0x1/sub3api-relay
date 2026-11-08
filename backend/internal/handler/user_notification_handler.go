package handler

import (
	"strconv"

	"github.com/yuchen0x1/sub3api-relay/internal/handler/dto"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	middleware2 "github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	"github.com/gin-gonic/gin"
)

// UserNotificationHandler 处理用户站内通知操作。
type UserNotificationHandler struct {
	notificationService *service.UserNotificationService
}

// NewUserNotificationHandler 创建站内通知处理器。
func NewUserNotificationHandler(notificationService *service.UserNotificationService) *UserNotificationHandler {
	return &UserNotificationHandler{notificationService: notificationService}
}

// List 列出当前用户的通知。
// GET /api/v1/notifications
func (h *UserNotificationHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	filter := &service.UserNotificationFilter{
		UnreadOnly: parseBoolQuery(c.Query("unread_only")),
		Type:       c.Query("type"),
	}
	if afterID := c.Query("after_id"); afterID != "" {
		if v, err := strconv.ParseInt(afterID, 10, 64); err == nil {
			filter.AfterID = v
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filter.Limit = v
		}
	}

	result, err := h.notificationService.List(c.Request.Context(), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserNotification, 0, len(result.Items))
	for i := range result.Items {
		out = append(out, *dto.UserNotificationFromService(&result.Items[i]))
	}
	response.Success(c, gin.H{
		"items":         out,
		"total":         result.Total,
		"next_after_id": result.NextAfterID,
	})
}

// CountUnread 返回当前用户未读通知数。
// GET /api/v1/notifications/unread-count
func (h *UserNotificationHandler) CountUnread(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	count, err := h.notificationService.CountUnread(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"count": count})
}

// MarkRead 标记单条通知为已读。
// POST /api/v1/notifications/:id/read
func (h *UserNotificationHandler) MarkRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid notification ID")
		return
	}
	if err := h.notificationService.MarkRead(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

// MarkAllRead 将所有通知标记为已读。
// POST /api/v1/notifications/read-all
func (h *UserNotificationHandler) MarkAllRead(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	if err := h.notificationService.MarkAllRead(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}
