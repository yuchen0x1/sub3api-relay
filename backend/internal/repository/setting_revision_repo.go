package repository

import (
	"context"
	"fmt"

	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	dbsettingrevision "github.com/yuchen0x1/sub3api-relay/ent/settingrevision"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// settingRevisionRepository 配置版本快照的 ent 实现。
type settingRevisionRepository struct {
	client *dbent.Client
}

// NewSettingRevisionRepository 创建配置版本存储。
func NewSettingRevisionRepository(client *dbent.Client) service.SettingRevisionRepository {
	return &settingRevisionRepository{client: client}
}

func (r *settingRevisionRepository) Create(ctx context.Context, snapshot string, changedBy int64, note string) (*service.SettingRevisionEntry, error) {
	created, err := r.client.SettingRevision.Create().
		SetSnapshot(snapshot).
		SetChangedBy(changedBy).
		SetNote(note).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create setting revision: %w", err)
	}
	return settingRevisionEntityToService(created), nil
}

func (r *settingRevisionRepository) GetByID(ctx context.Context, id int64) (*service.SettingRevisionEntry, error) {
	m, err := r.client.SettingRevision.Query().
		Where(dbsettingrevision.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSettingRevisionNotFound, nil)
	}
	return settingRevisionEntityToService(m), nil
}

func (r *settingRevisionRepository) List(ctx context.Context, limit, offset int) ([]*service.SettingRevisionEntry, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	q := r.client.SettingRevision.Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count setting revisions: %w", err)
	}
	rows, err := q.
		Order(dbent.Desc(dbsettingrevision.FieldID)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list setting revisions: %w", err)
	}
	entries := make([]*service.SettingRevisionEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, settingRevisionEntityToService(row))
	}
	return entries, int64(total), nil
}

func settingRevisionEntityToService(m *dbent.SettingRevision) *service.SettingRevisionEntry {
	if m == nil {
		return nil
	}
	return &service.SettingRevisionEntry{
		ID:        m.ID,
		Snapshot:  m.Snapshot,
		ChangedBy: m.ChangedBy,
		Note:      m.Note,
		CreatedAt: m.CreatedAt,
	}
}
