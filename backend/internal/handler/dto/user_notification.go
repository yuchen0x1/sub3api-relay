package dto

import (
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// UserNotification 用户站内通知的 DTO（对外 JSON 结构）。
type UserNotification struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// UserNotificationFromService 将 service 层结构转换为 DTO。
func UserNotificationFromService(n *service.UserNotification) *UserNotification {
	if n == nil {
		return nil
	}
	return &UserNotification{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Metadata:  n.Metadata,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}
