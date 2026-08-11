package repository

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	"github.com/yuchen0x1/sub3api-relay/ent/setting"
	dbuser "github.com/yuchen0x1/sub3api-relay/ent/user"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

const (
	simpleModeAdminConcurrencyUpgradeKey = "simple_mode_admin_concurrency_upgraded_unlimited_175"
	simpleModeTargetAdminConcurrency     = 0
)

func ensureSimpleModeAdminConcurrency(ctx context.Context, client *dbent.Client) error {
	if client == nil {
		return fmt.Errorf("nil ent client")
	}

	upgraded, err := client.Setting.Query().Where(setting.KeyEQ(simpleModeAdminConcurrencyUpgradeKey)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check admin concurrency upgrade marker: %w", err)
	}
	if upgraded {
		return nil
	}

	if _, err := client.User.Update().
		Where(
			dbuser.RoleEQ(service.RoleAdmin),
			dbuser.ConcurrencyGT(0),
		).
		SetConcurrency(simpleModeTargetAdminConcurrency).
		Save(ctx); err != nil {
		return fmt.Errorf("upgrade simple mode admin concurrency: %w", err)
	}

	now := time.Now()
	if err := client.Setting.Create().
		SetKey(simpleModeAdminConcurrencyUpgradeKey).
		SetValue(now.Format(time.RFC3339)).
		SetUpdatedAt(now).
		OnConflictColumns(setting.FieldKey).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return fmt.Errorf("persist admin concurrency upgrade marker: %w", err)
	}

	return nil
}
