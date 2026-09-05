package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// Key 级 QPS 计数器 Redis 实现。
//
// 设计说明：
//   - key 形式：qps:k:{apiKeyID}:{second}
//   - 时间来源：rdb.Time()（Redis 服务端时间），避免多实例时钟漂移。
//   - 原子操作：TxPipeline (MULTI/EXEC) 执行 INCR+EXPIRE，兼容 Redis Cluster。
//   - TTL：2s，覆盖当前秒窗口 + 冗余。
//   - 1 秒固定窗口在窗口边界处允许最多 2 倍突发，与分钟级 RPM 计数同一取舍。
const (
	apiKeyQPSKeyPrefix = "qps:k:"

	apiKeyQPSKeyTTL = 2 * time.Second
)

type apiKeyQPSCacheImpl struct {
	rdb *redis.Client
}

// NewAPIKeyQPSCache 创建 Key 级 QPS 计数器。
func NewAPIKeyQPSCache(rdb *redis.Client) service.APIKeyQPSCache {
	return &apiKeyQPSCacheImpl{rdb: rdb}
}

// secondTS 获取当前 Redis 服务端秒级时间戳。
func (c *apiKeyQPSCacheImpl) secondTS(ctx context.Context) (int64, error) {
	t, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return 0, fmt.Errorf("redis TIME: %w", err)
	}
	return t.Unix(), nil
}

// IncrementKeyQPS 递增 Key 当前秒计数并返回最新值。
func (c *apiKeyQPSCacheImpl) IncrementKeyQPS(ctx context.Context, apiKeyID int64) (int, error) {
	second, err := c.secondTS(ctx)
	if err != nil {
		return 0, err
	}
	key := fmt.Sprintf("%s%d:%d", apiKeyQPSKeyPrefix, apiKeyID, second)
	pipe := c.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, apiKeyQPSKeyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("api key qps increment: %w", err)
	}
	return int(incr.Val()), nil
}
