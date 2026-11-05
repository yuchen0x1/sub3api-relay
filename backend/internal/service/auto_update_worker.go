package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/sysutil"
)

const (
	// autoUpdateOperationID 自动更新使用的系统操作锁标识，与手动更新/回滚/重启共享全局锁
	autoUpdateOperationID = "auto_update"
	// autoUpdateInitialDelay 服务启动后等待一段时间再做首次检查，避免与启动流程抢资源
	autoUpdateInitialDelay = 1 * time.Minute
	// autoUpdateCheckTimeout 单次检查+下载+替换的总体超时（与手动更新的 15 分钟保持一致）
	autoUpdateCheckTimeout = 15 * time.Minute
	// autoUpdateDefaultIntervalHours 默认自动检查间隔（小时）
	autoUpdateDefaultIntervalHours = 12
)

// AutoUpdateWorker 定期检测 GitHub Release 新版本，发现后自动下载替换并重启服务。
// 仅在 release 构建且 update.auto_enabled=true 时运行；
// 通过全局系统操作锁与手动更新/回滚/重启互斥，避免并发替换二进制。
type AutoUpdateWorker struct {
	updateSvc *UpdateService
	lockSvc   *SystemOperationLockService
	cfg       *config.Config
	interval  time.Duration

	mu      sync.Mutex
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewAutoUpdateWorker 创建自动更新调度器
func NewAutoUpdateWorker(updateSvc *UpdateService, lockSvc *SystemOperationLockService, cfg *config.Config) *AutoUpdateWorker {
	interval := time.Duration(autoUpdateDefaultIntervalHours) * time.Hour
	if cfg != nil && cfg.Update.AutoIntervalHours > 0 {
		interval = time.Duration(cfg.Update.AutoIntervalHours) * time.Hour
	}
	return &AutoUpdateWorker{
		updateSvc: updateSvc,
		lockSvc:   lockSvc,
		cfg:       cfg,
		interval:  interval,
	}
}

// ProvideAutoUpdateWorker 创建并启动自动更新调度器
func ProvideAutoUpdateWorker(updateSvc *UpdateService, lockSvc *SystemOperationLockService, cfg *config.Config) *AutoUpdateWorker {
	worker := NewAutoUpdateWorker(updateSvc, lockSvc, cfg)
	worker.Start()
	return worker
}

// Start 启动调度循环，仅在 release 构建且开启自动更新时真正运行
func (w *AutoUpdateWorker) Start() {
	if w == nil || w.updateSvc == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started || w.stopped {
		return
	}
	if w.updateSvc.buildType != "release" {
		slog.Info("auto_update: worker disabled, build type is not release", "build_type", w.updateSvc.buildType)
		w.stopped = true
		return
	}
	if w.cfg == nil || !w.cfg.Update.AutoEnabled {
		slog.Info("auto_update: worker disabled, update.auto_enabled=false")
		w.stopped = true
		return
	}
	w.started = true
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(1)
	slog.Info("auto_update: worker started", "interval_hours", w.interval.Hours())
	go w.runLoop(ctx)
}

// Stop 停止调度循环
func (w *AutoUpdateWorker) Stop() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	if w.cancel != nil {
		w.cancel()
	}
	w.mu.Unlock()
	w.wg.Wait()
}

func (w *AutoUpdateWorker) runLoop(ctx context.Context) {
	defer w.wg.Done()

	first := time.NewTimer(autoUpdateInitialDelay)
	defer first.Stop()
	select {
	case <-ctx.Done():
		return
	case <-first.C:
		w.checkAndUpdate()
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkAndUpdate()
		}
	}
}

func (w *AutoUpdateWorker) checkAndUpdate() {
	if w == nil || w.updateSvc == nil || w.lockSvc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), autoUpdateCheckTimeout)
	defer cancel()

	// Resolve the fixed GitHub update channel for this check and include it in logs.
	channel := w.resolveChannel(ctx)
	slog.Info("auto_update: checking for updates", "channel", string(channel))

	info, err := w.updateSvc.CheckUpdate(ctx, true)
	if err != nil {
		slog.Warn("auto_update: check update failed", "channel", string(channel), "error", err)
		return
	}
	if info == nil || !info.HasUpdate {
		return
	}
	if info.BuildType != "release" {
		slog.Info("auto_update: update available but build type is not release, skip",
			"channel", string(channel), "latest_version", info.LatestVersion)
		return
	}

	lock, err := w.lockSvc.Acquire(ctx, autoUpdateOperationID)
	if err != nil {
		slog.Warn("auto_update: system operation busy, skip", "error", err)
		return
	}

	slog.Info("auto_update: new release detected, applying update",
		"channel", string(channel), "current_version", info.CurrentVersion, "latest_version", info.LatestVersion)
	if err := w.updateSvc.PerformUpdate(ctx); err != nil {
		if errors.Is(err, ErrNoUpdateAvailable) {
			_ = w.lockSvc.Release(context.Background(), lock, true, "")
			return
		}
		_ = w.lockSvc.Release(context.Background(), lock, false, "AUTO_UPDATE_FAILED")
		slog.Error("auto_update: apply update failed", "channel", string(channel), "error", err)
		return
	}

	// 先释放锁再重启，避免重启后 TTL 内手动操作被误判为 busy
	_ = w.lockSvc.Release(context.Background(), lock, true, "")
	slog.Info("auto_update: update applied, restarting service",
		"channel", string(channel), "latest_version", info.LatestVersion)
	sysutil.RestartServiceAsync()
}

// resolveChannel 解析本轮更新使用的通道，任何异常均兜底为 github
func (w *AutoUpdateWorker) resolveChannel(ctx context.Context) UpdateChannel {
	channel, err := ResolveUpdateChannel(ctx)
	if err != nil || channel == "" {
		return UpdateChannelGitHub
	}
	return channel
}
