package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	dbgatewaybodylog "github.com/yuchen0x1/sub3api-relay/ent/gatewaybodylog"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// gatewayBodyLogRepository 请求体日志存储的 ent 实现。
type gatewayBodyLogRepository struct {
	client *dbent.Client
}

// NewGatewayBodyLogRepository 创建请求体日志存储。
func NewGatewayBodyLogRepository(client *dbent.Client) service.GatewayBodyLogRepository {
	return &gatewayBodyLogRepository{client: client}
}

func (r *gatewayBodyLogRepository) Create(ctx context.Context, requestID, method, path string, apiKeyID, userID int64, model, body string) (*service.GatewayBodyLogEntry, error) {
	created, err := r.client.GatewayBodyLog.Create().
		SetRequestID(requestID).
		SetMethod(method).
		SetPath(path).
		SetAPIKeyID(apiKeyID).
		SetUserID(userID).
		SetModel(model).
		SetBody(body).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create gateway body log: %w", err)
	}
	return gatewayBodyLogEntityToService(created), nil
}

func (r *gatewayBodyLogRepository) GetByID(ctx context.Context, id int64) (*service.GatewayBodyLogEntry, error) {
	m, err := r.client.GatewayBodyLog.Query().
		Where(dbgatewaybodylog.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGatewayBodyLogNotFound, nil)
	}
	return gatewayBodyLogEntityToService(m), nil
}

func (r *gatewayBodyLogRepository) List(ctx context.Context, filter service.GatewayBodyLogFilter) ([]*service.GatewayBodyLogEntry, int64, error) {
	q := r.client.GatewayBodyLog.Query()
	if filter.RequestID != "" {
		q = q.Where(dbgatewaybodylog.RequestIDEQ(filter.RequestID))
	}
	if filter.APIKeyID > 0 {
		q = q.Where(dbgatewaybodylog.APIKeyIDEQ(filter.APIKeyID))
	}
	if filter.UserID > 0 {
		q = q.Where(dbgatewaybodylog.UserIDEQ(filter.UserID))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count gateway body logs: %w", err)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	rows, err := q.
		Order(dbent.Desc(dbgatewaybodylog.FieldCreatedAt), dbent.Desc(dbgatewaybodylog.FieldID)).
		Limit(limit).
		Offset(filter.Offset).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list gateway body logs: %w", err)
	}
	entries := make([]*service.GatewayBodyLogEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, gatewayBodyLogEntityToService(row))
	}
	return entries, int64(total), nil
}

func (r *gatewayBodyLogRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	count, err := r.client.GatewayBodyLog.Delete().
		Where(dbgatewaybodylog.CreatedAtLT(before)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete old gateway body logs: %w", err)
	}
	return int64(count), nil
}

func gatewayBodyLogEntityToService(m *dbent.GatewayBodyLog) *service.GatewayBodyLogEntry {
	if m == nil {
		return nil
	}
	return &service.GatewayBodyLogEntry{
		ID:        m.ID,
		RequestID: m.RequestID,
		Method:    m.Method,
		Path:      m.Path,
		APIKeyID:  m.APIKeyID,
		UserID:    m.UserID,
		Model:     m.Model,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
	}
}
