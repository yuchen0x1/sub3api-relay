package service

import (
	"context"
	"log/slog"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
)

// ErrAPIKeyQPSLimited 表示该 API Key 在当前秒内请求数超过 qps_limit。
var ErrAPIKeyQPSLimited = infraerrors.TooManyRequests("API_KEY_QPS_LIMITED", "api key QPS limit exceeded")

// APIKeyQPSCache Key 级 QPS 计数器接口。
//
// 与 UserRPMCache 的区别：按单个 API Key 聚合、窗口为 1 秒（RPM 按 (user[,group])
// 聚合、窗口为 1 分钟）。key 形如 qps:k:{apiKeyID}:{second}。
type APIKeyQPSCache interface {
	// IncrementKeyQPS 原子递增该 Key 当前秒的计数并返回最新值。
	IncrementKeyQPS(ctx context.Context, apiKeyID int64) (count int, err error)
}

// MaxAPIKeyQPSLimit qps_limit 的上限，防止误配置出无意义的超大值。
const MaxAPIKeyQPSLimit = 100000

// normalizeAPIKeyQPSLimit 把 qps_limit 归一到 [0, MaxAPIKeyQPSLimit]。
func normalizeAPIKeyQPSLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	if limit > MaxAPIKeyQPSLimit {
		return MaxAPIKeyQPSLimit
	}
	return limit
}

// SetQPSCounterCache 注入 Key 级 QPS 计数器实现；nil 表示禁用 QPS 限流。
func (s *APIKeyService) SetQPSCounterCache(cache APIKeyQPSCache) {
	if s == nil {
		return
	}
	s.qpsCounter = cache
}

// CheckAndIncrementKeyQPS 递增该 Key 的秒级计数；超过 qps_limit 时返回
// ErrAPIKeyQPSLimited。Redis 不可用或未配置计数器时放行（fail-open）：
// QPS 是保护性限流而非安全边界，可用性优先。
func (s *APIKeyService) CheckAndIncrementKeyQPS(ctx context.Context, apiKey *APIKey) error {
	if s == nil || apiKey == nil || apiKey.QPSLimit <= 0 || s.qpsCounter == nil {
		return nil
	}
	count, err := s.qpsCounter.IncrementKeyQPS(ctx, apiKey.ID)
	if err != nil {
		slog.Debug("api key qps counter unavailable, allowing request", "api_key_id", apiKey.ID, "error", err)
		return nil
	}
	if count > apiKey.QPSLimit {
		return ErrAPIKeyQPSLimited
	}
	return nil
}
