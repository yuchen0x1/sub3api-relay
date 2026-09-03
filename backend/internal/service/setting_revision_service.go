package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
)

// ErrSettingRevisionNotFound 配置版本不存在。
var ErrSettingRevisionNotFound = infraerrors.NotFound("SETTING_REVISION_NOT_FOUND", "setting revision not found")

// SettingRevisionEntry 配置版本快照。
type SettingRevisionEntry struct {
	ID        int64     `json:"id"`
	Snapshot  string    `json:"-"`
	ChangedBy int64     `json:"changed_by"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// SettingRevisionRepository 配置版本存储接口。
type SettingRevisionRepository interface {
	Create(ctx context.Context, snapshot string, changedBy int64, note string) (*SettingRevisionEntry, error)
	GetByID(ctx context.Context, id int64) (*SettingRevisionEntry, error)
	List(ctx context.Context, limit, offset int) ([]*SettingRevisionEntry, int64, error)
}

// SettingRevisionLimit 配置版本最多保留条数。
const SettingRevisionLimit = 50

// SetSettingRevisionRepository 注入配置版本存储；nil 表示停用版本记录。
func (s *SettingService) SetSettingRevisionRepository(repo SettingRevisionRepository) {
	if s == nil {
		return
	}
	s.revisionRepo = repo
}

// RecordSettingRevision 在设置变更前调用：把当前全量键值快照存为一条版本。
// 记录失败仅记日志，不阻塞设置保存。
func (s *SettingService) RecordSettingRevision(ctx context.Context, changedBy int64, note string) {
	if s == nil || s.revisionRepo == nil || s.settingRepo == nil {
		return
	}
	all, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		slog.Warn("setting revision snapshot failed", "error", err)
		return
	}
	raw, err := json.Marshal(all)
	if err != nil {
		slog.Warn("setting revision marshal failed", "error", err)
		return
	}
	rev, err := s.revisionRepo.Create(ctx, string(raw), changedBy, note)
	if err != nil {
		slog.Warn("setting revision persist failed", "error", err)
	}
	_ = rev
	// 历史不做自动裁剪：设置保存是低频管理操作，单条快照仅含设置键值，
	// 体量可控；如需清理可由数据管理路径统一处理。
}

// ListSettingRevisions 分页列出配置版本（不含快照正文）。
func (s *SettingService) ListSettingRevisions(ctx context.Context, limit, offset int) ([]*SettingRevisionEntry, int64, error) {
	if s == nil || s.revisionRepo == nil {
		return nil, 0, ErrSettingRevisionNotFound
	}
	return s.revisionRepo.List(ctx, limit, offset)
}

// RollbackSettingRevision 把设置整体回滚到指定版本（快照的全量键值覆盖当前值）。
func (s *SettingService) RollbackSettingRevision(ctx context.Context, id int64) error {
	if s == nil || s.revisionRepo == nil || s.settingRepo == nil {
		return ErrSettingRevisionNotFound
	}
	rev, err := s.revisionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	var snapshot map[string]string
	if err := json.Unmarshal([]byte(rev.Snapshot), &snapshot); err != nil {
		return infraerrors.BadRequest("SETTING_REVISION_INVALID_SNAPSHOT", "setting revision snapshot is corrupted")
	}
	if err := s.settingRepo.SetMultiple(ctx, snapshot); err != nil {
		return err
	}
	// 回滚后从存储重建进程内缓存。
	stored, err := s.GetAllSettings(ctx)
	if err == nil && stored != nil {
		s.refreshCachedSettings(stored)
	}
	return nil
}
