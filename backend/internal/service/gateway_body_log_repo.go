package service

import (
	"context"
	"io"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
)

// ErrGatewayBodyLogNotFound 请求体日志不存在。
var ErrGatewayBodyLogNotFound = infraerrors.NotFound("GATEWAY_BODY_LOG_NOT_FOUND", "gateway body log not found")

// GatewayBodyLogEntry 请求体日志条目（与应用层传递的统一形态）。
type GatewayBodyLogEntry struct {
	ID        int64     `json:"id"`
	RequestID string    `json:"request_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	APIKeyID  int64     `json:"api_key_id"`
	UserID    int64     `json:"user_id"`
	Model     string    `json:"model"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// GatewayBodyLogFilter 列表过滤条件（零值 = 不过滤）。
type GatewayBodyLogFilter struct {
	RequestID string
	APIKeyID  int64
	UserID    int64
	Limit     int
	Offset    int
}

// GatewayBodyLogRepository 请求体日志存储接口。
type GatewayBodyLogRepository interface {
	Create(ctx context.Context, requestID, method, path string, apiKeyID, userID int64, model, body string) (*GatewayBodyLogEntry, error)
	GetByID(ctx context.Context, id int64) (*GatewayBodyLogEntry, error)
	List(ctx context.Context, filter GatewayBodyLogFilter) ([]*GatewayBodyLogEntry, int64, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// ioLimit 读取最多 n 字节。
func ioLimit(r io.Reader, n int64) io.Reader {
	return io.LimitReader(r, n)
}
