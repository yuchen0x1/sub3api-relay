package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// GatewayBodyLogService 记录/查看/重放网关请求体。
//
// 记录由设置 gateway_body_log_enabled 控制（默认关闭，60s 缓存），写入为
// fire-and-forget：中间件只负责把请求体留在内存里，落库在独立 goroutine。
// 重放通过环回 HTTP 请求把存储的请求体重新打回本机网关端点，完整走一遍
// 鉴权/调度/转发链路。
type GatewayBodyLogService struct {
	repo       GatewayBodyLogRepository
	apiKeyRepo APIKeyRepository
	settingSvc *SettingService
	cfg        ServerAddressConfig
	client     *http.Client
}

// ServerAddressConfig 提供本机网关监听地址（用于环回重放），解耦 *config.Config。
type ServerAddressConfig interface {
	Address() string
}

// MaxGatewayBodyLogBytes 单条请求体落库上限；超长截断并追加省略标记。
const MaxGatewayBodyLogBytes = 256 * 1024

// GatewayBodyLogReplayTimeout 环回重放的超时时间。
const GatewayBodyLogReplayTimeout = 120 * time.Second

// NewGatewayBodyLogService 创建服务；任一依赖为 nil 时功能整体停用。
func NewGatewayBodyLogService(repo GatewayBodyLogRepository, apiKeyRepo APIKeyRepository, settingSvc *SettingService, serverCfg ServerAddressConfig) *GatewayBodyLogService {
	return &GatewayBodyLogService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
		settingSvc: settingSvc,
		cfg:        serverCfg,
		client:     &http.Client{Timeout: GatewayBodyLogReplayTimeout},
	}
}

// IsEnabled 报告请求体记录是否开启（设置中心开关，默认关）。
func (s *GatewayBodyLogService) IsEnabled(ctx context.Context) bool {
	if s == nil || s.settingSvc == nil {
		return false
	}
	return s.settingSvc.GetGatewayBodyLogEnabled(ctx)
}

// Record 异步保存一条请求体日志。body 会被复制，调用方可以安全复用原切片。
func (s *GatewayBodyLogService) Record(ctx context.Context, entry *GatewayBodyLogEntry) {
	if s == nil || s.repo == nil || !s.IsEnabled(ctx) {
		return
	}
	stored := make([]byte, len(entry.Body))
	copy(stored, entry.Body)
	truncated := false
	if len(stored) > MaxGatewayBodyLogBytes {
		stored = stored[:MaxGatewayBodyLogBytes]
		truncated = true
	}
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		body := string(stored)
		if truncated {
			body += "\n...[truncated]"
		}
		_, err := s.repo.Create(bgCtx, entry.RequestID, entry.Method, entry.Path, entry.APIKeyID, entry.UserID, entry.Model, body)
		if err != nil {
			slog.Debug("gateway body log persist failed", "request_id", entry.RequestID, "error", err)
		}
	}()
}

// Get 按 ID 获取请求体日志。
func (s *GatewayBodyLogService) Get(ctx context.Context, id int64) (*GatewayBodyLogEntry, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGatewayBodyLogNotFound
	}
	return s.repo.GetByID(ctx, id)
}

// List 按过滤条件分页列出请求体日志。
func (s *GatewayBodyLogService) List(ctx context.Context, filter GatewayBodyLogFilter) ([]*GatewayBodyLogEntry, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, ErrGatewayBodyLogNotFound
	}
	return s.repo.List(ctx, filter)
}

// GatewayBodyLogReplayResult 环回重放的结果。
type GatewayBodyLogReplayResult struct {
	Status  int    `json:"status"`
	Body    string `json:"body"`
	Elapsed int64  `json:"elapsed_ms"`
}

// Replay 把存储的请求体通过环回 HTTP 请求重放至本机网关端点。
// 使用原 API Key 明文鉴权（api_keys.key 本就保存明文 key，用于认证比对）。
func (s *GatewayBodyLogService) Replay(ctx context.Context, id int64) (*GatewayBodyLogReplayResult, error) {
	if s == nil || s.repo == nil || s.cfg == nil || s.cfg.Address() == "" {
		return nil, fmt.Errorf("gateway body replay is not available")
	}
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	rawKey := ""
	if s.apiKeyRepo != nil && entry.APIKeyID > 0 {
		if key, kerr := s.apiKeyRepo.GetByID(ctx, entry.APIKeyID); kerr == nil && key != nil {
			rawKey = key.Key
		}
	}
	if strings.TrimSpace(rawKey) == "" {
		return nil, fmt.Errorf("original API key no longer exists; cannot replay")
	}

	endpoint := "http://" + s.cfg.Address() + entry.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte(entry.Body)))
	if err != nil {
		return nil, fmt.Errorf("build replay request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawKey)

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("replay request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	buf := &bytes.Buffer{}
	_, _ = buf.ReadFrom(ioLimit(resp.Body, MaxGatewayBodyLogBytes))
	return &GatewayBodyLogReplayResult{
		Status:  resp.StatusCode,
		Body:    buf.String(),
		Elapsed: time.Since(start).Milliseconds(),
	}, nil
}
