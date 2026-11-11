package service

import (
	"context"
	"errors"
	"time"
)

// 站内通知（user notifications）的 service 层传输结构与数据访问端口。
//
// 通知中心将公告(announcements, 面向群体广播) 与个性化站内通知
// (user_notifications, 面向单个用户的事件) 统一在 /notifications 呈现。
// 本文件定义后者所需的 model 与 repository 端口。

// 站内通知类型。
const (
	NotificationTypeSystem          = "system"
	NotificationTypeBalanceLow      = "balance_low"
	NotificationTypeAccountQuota    = "account_quota"
	NotificationTypePaymentReceived = "payment_received"
	NotificationTypeKeyRotated      = "key_rotated"
)

// ErrUserNotificationNilInput 输入为空。
var ErrUserNotificationNilInput = errors.New("user notification nil input")

// ErrUserNotificationInvalidUser 用户 ID 非法。
var ErrUserNotificationInvalidUser = errors.New("user notification invalid user")

// UserNotification 单条站内通知记录。
type UserNotification struct {
	ID        int64
	UserID    int64
	Type      string
	Title     string
	Body      string
	Metadata  map[string]any
	ReadAt    *time.Time
	CreatedAt time.Time
}

// UserNotificationCreateInput 创建站内通知的输入。
type UserNotificationCreateInput struct {
	UserID   int64
	Type     string
	Title    string
	Body     string
	Metadata map[string]any
}

// UserNotificationFilter 查询站内通知的过滤条件。
type UserNotificationFilter struct {
	UserID     int64
	Type       string // 空 = 全部类型
	UnreadOnly bool
	// 分页（游标式，按 id 倒序）。
	AfterID int64 // 返回 id < AfterID 的记录；0 = 从头开始
	Limit   int   // <=0 时使用默认值
}

// UserNotificationList 分页结果。
type UserNotificationList struct {
	Items []UserNotification
	// NextAfterID 供下一次游标分页；0 表示没有更多。
	NextAfterID int64
	Total       int64
}

// UserNotificationRepository 定义 service 层所需的站内通知数据访问端口。
type UserNotificationRepository interface {
	// Create 落库一条站内通知并回填 ID/CreatedAt。
	Create(ctx context.Context, input *UserNotificationCreateInput) (*UserNotification, error)
	// List 按过滤条件查询当前用户的通知（倒序分页）。
	List(ctx context.Context, filter *UserNotificationFilter) (*UserNotificationList, error)
	// CountUnread 返回当前用户未读通知数。
	CountUnread(ctx context.Context, userID int64) (int64, error)
	// MarkRead 将指定通知标记为已读（仅限属于该用户的通知）。
	MarkRead(ctx context.Context, userID int64, id int64) error
	// MarkAllRead 将该用户所有通知标记为已读。
	MarkAllRead(ctx context.Context, userID int64) error
}

// UserNotificationService 提供站内通知的读写能力。
type UserNotificationService struct {
	repo UserNotificationRepository
}

// NewUserNotificationService 创建站内通知服务。
func NewUserNotificationService(repo UserNotificationRepository) *UserNotificationService {
	return &UserNotificationService{repo: repo}
}

// Create 创建一条站内通知。
func (s *UserNotificationService) Create(ctx context.Context, input *UserNotificationCreateInput) (*UserNotification, error) {
	if input == nil {
		return nil, ErrUserNotificationNilInput
	}
	if input.UserID <= 0 {
		return nil, ErrUserNotificationInvalidUser
	}
	typ := input.Type
	if typ == "" {
		typ = NotificationTypeSystem
	}
	meta := input.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	return s.repo.Create(ctx, &UserNotificationCreateInput{
		UserID:   input.UserID,
		Type:     typ,
		Title:    input.Title,
		Body:     input.Body,
		Metadata: meta,
	})
}

// List 查询当前用户的通知。
func (s *UserNotificationService) List(ctx context.Context, userID int64, filter *UserNotificationFilter) (*UserNotificationList, error) {
	if filter == nil {
		filter = &UserNotificationFilter{}
	}
	filter.UserID = userID
	return s.repo.List(ctx, filter)
}

// CountUnread 返回当前用户未读通知数。
func (s *UserNotificationService) CountUnread(ctx context.Context, userID int64) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead 标记单条为已读。
func (s *UserNotificationService) MarkRead(ctx context.Context, userID int64, id int64) error {
	return s.repo.MarkRead(ctx, userID, id)
}

// MarkAllRead 全部标记为已读。
func (s *UserNotificationService) MarkAllRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllRead(ctx, userID)
}
