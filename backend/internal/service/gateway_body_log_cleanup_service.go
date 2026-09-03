package service

import (
	"context"
	"sync"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

// BodyLogRetentionProvider 提供请求体日志保留天数（0 = 永久保留），解耦 *SettingService。
type BodyLogRetentionProvider interface {
	GetGatewayBodyLogRetentionDays(ctx context.Context) int
}

// GatewayBodyLogCleanupService 定期删除超过保留期的请求体日志，避免表无限膨胀。
// 保留天数来自设置 gateway_body_log_retention_days（默认 7，0 = 永久保留）。
type GatewayBodyLogCleanupService struct {
	repo      GatewayBodyLogRepository
	retention BodyLogRetentionProvider
	interval  time.Duration

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
}

// NewGatewayBodyLogCleanupService 创建清理服务；repo 或 retention 为 nil 时不启动。
func NewGatewayBodyLogCleanupService(repo GatewayBodyLogRepository, retention BodyLogRetentionProvider) *GatewayBodyLogCleanupService {
	return &GatewayBodyLogCleanupService{
		repo:      repo,
		retention: retention,
		interval:  10 * time.Minute,
		stopCh:    make(chan struct{}),
	}
}

func (s *GatewayBodyLogCleanupService) Start() {
	if s == nil || s.repo == nil || s.retention == nil {
		return
	}
	s.startOnce.Do(func() {
		logger.LegacyPrintf("service.gateway_body_log_cleanup", "[GatewayBodyLogCleanup] started interval=%s", s.interval)
		go s.runLoop()
	})
}

func (s *GatewayBodyLogCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		// stopCh 由构造函数初始化；零值服务（未经过构造函数）没有 channel 可关。
		if s.stopCh != nil {
			close(s.stopCh)
		}
		logger.LegacyPrintf("service.gateway_body_log_cleanup", "[GatewayBodyLogCleanup] stopped")
	})
}

func (s *GatewayBodyLogCleanupService) runLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 启动后先清理一轮，防止重启后积压。
	s.cleanupOnce()

	for {
		select {
		case <-ticker.C:
			s.cleanupOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *GatewayBodyLogCleanupService) cleanupOnce() {
	if s == nil || s.repo == nil || s.retention == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	retentionDays := s.retention.GetGatewayBodyLogRetentionDays(ctx)
	if retentionDays <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		logger.LegacyPrintf("service.gateway_body_log_cleanup", "[GatewayBodyLogCleanup] cleanup failed err=%v", err)
		return
	}
	if deleted > 0 {
		logger.LegacyPrintf("service.gateway_body_log_cleanup", "[GatewayBodyLogCleanup] cleaned expired body logs retention_days=%d count=%d", retentionDays, deleted)
	}
}
