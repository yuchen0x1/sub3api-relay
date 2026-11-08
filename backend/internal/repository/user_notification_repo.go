package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// userNotificationRepository 基于原生 SQL 实现 service.UserNotificationRepository。
type userNotificationRepository struct {
	db *sql.DB
}

// NewUserNotificationRepository 创建站内通知仓储。
func NewUserNotificationRepository(db *sql.DB) service.UserNotificationRepository {
	return &userNotificationRepository{db: db}
}

const defaultNotificationLimit = 20

func (r *userNotificationRepository) Create(ctx context.Context, input *service.UserNotificationCreateInput) (*service.UserNotification, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil user notification repository")
	}
	if input == nil {
		return nil, fmt.Errorf("nil input")
	}
	meta, err := json.Marshal(input.Metadata)
	if err != nil {
		meta = []byte("{}")
	}
	row := r.db.QueryRowContext(ctx, `
INSERT INTO user_notifications (user_id, type, title, body, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at`,
		input.UserID, input.Type, input.Title, input.Body, meta)

	n := &service.UserNotification{
		UserID:   input.UserID,
		Type:     input.Type,
		Title:    input.Title,
		Body:     input.Body,
		Metadata: input.Metadata,
	}
	if err := row.Scan(&n.ID, &n.CreatedAt); err != nil {
		return nil, err
	}
	return n, nil
}

func (r *userNotificationRepository) List(ctx context.Context, filter *service.UserNotificationFilter) (*service.UserNotificationList, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil user notification repository")
	}
	if filter == nil {
		filter = &service.UserNotificationFilter{}
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultNotificationLimit
	}
	if limit > 100 {
		limit = 100
	}

	// 先取满足条件的总条数。
	countQ := `SELECT COUNT(*) FROM user_notifications WHERE user_id = $1`
	countArgs := []any{filter.UserID}
	cond := ""
	if filter.Type != "" {
		cond += " AND type = $2"
		countArgs = append(countArgs, filter.Type)
	}
	if filter.UnreadOnly {
		cond += " AND read_at IS NULL"
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ+cond, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	// 分页查询（id 倒序，游标 AfterID 即"比该 id 更早"）。
	// 取 limit+1 以判断是否还有下一页。
	selQ := `
SELECT id, user_id, type, title, body, metadata, read_at, created_at
FROM user_notifications
WHERE user_id = $1`
	args := []any{filter.UserID}
	if filter.Type != "" {
		args = append(args, filter.Type)
		selQ += " AND type = $" + fmt.Sprintf("%d", len(args))
	}
	if filter.UnreadOnly {
		selQ += " AND read_at IS NULL"
	}
	if filter.AfterID > 0 {
		args = append(args, filter.AfterID)
		selQ += " AND id < $" + fmt.Sprintf("%d", len(args))
	}
	selQ += ` ORDER BY id DESC LIMIT ` + fmt.Sprintf("%d", limit+1)

	rows, err := r.db.QueryContext(ctx, selQ, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.UserNotification, 0, limit)
	for rows.Next() {
		var n service.UserNotification
		var meta []byte
		var readAt sql.NullTime
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &meta, &readAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		if readAt.Valid {
			t := readAt.Time
			n.ReadAt = &t
		}
		n.Metadata = decodeJSONMap(meta)
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &service.UserNotificationList{Items: items, Total: total}
	if len(items) > limit {
		result.Items = items[:limit]
		result.NextAfterID = items[limit-1].ID
	}
	return result, nil
}

func (r *userNotificationRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil user notification repository")
	}
	var count int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *userNotificationRepository) MarkRead(ctx context.Context, userID int64, id int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil user notification repository")
	}
	if _, err := r.db.ExecContext(ctx,
		`UPDATE user_notifications SET read_at = NOW() WHERE user_id = $1 AND id = $2 AND read_at IS NULL`,
		userID, id); err != nil {
		return err
	}
	return nil
}

func (r *userNotificationRepository) MarkAllRead(ctx context.Context, userID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil user notification repository")
	}
	if _, err := r.db.ExecContext(ctx,
		`UPDATE user_notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID); err != nil {
		return err
	}
	return nil
}

// decodeJSONMap 将数据库返回的 JSONB 字节解析为 map，失败时返回空 map。
func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}
