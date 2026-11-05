//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type createLimitRepoStub struct {
	APIKeyRepository
	count int64
	err   error
}

func (s *createLimitRepoStub) CountByUserID(context.Context, int64) (int64, error) {
	return s.count, s.err
}

type createLimitCacheStub struct {
	APIKeyCache
	count int
	err   error
}

func (s *createLimitCacheStub) GetCreateSuccessCount(context.Context, int64) (int, error) {
	return s.count, s.err
}

func TestAPIKeyService_CheckAPIKeyCreateLimits(t *testing.T) {
	ctx := context.Background()

	t.Run("blocks at configured active key count", func(t *testing.T) {
		settings := NewSettingService(&settingRepoStub{values: map[string]string{
			SettingKeyAPIKeyMaxActivePerUser: "2",
		}}, nil)
		svc := &APIKeyService{
			apiKeyRepo:     &createLimitRepoStub{count: 2},
			settingService: settings,
		}

		err := svc.checkAPIKeyCreateLimits(ctx, 7)
		require.ErrorIs(t, err, ErrAPIKeyCountLimit)
	})

	t.Run("blocks at configured hourly creation count", func(t *testing.T) {
		settings := NewSettingService(&settingRepoStub{values: map[string]string{
			SettingKeyAPIKeyCreateHourlyLimit: "3",
		}}, nil)
		svc := &APIKeyService{
			apiKeyRepo:     &createLimitRepoStub{},
			cache:          &createLimitCacheStub{count: 3},
			settingService: settings,
		}

		err := svc.checkAPIKeyCreateLimits(ctx, 7)
		require.ErrorIs(t, err, ErrAPIKeyCreateLimited)
	})

	t.Run("zero disables both limits", func(t *testing.T) {
		settings := NewSettingService(&settingRepoStub{values: map[string]string{
			SettingKeyAPIKeyMaxActivePerUser:  "0",
			SettingKeyAPIKeyCreateHourlyLimit: "0",
		}}, nil)
		repo := &createLimitRepoStub{count: 1000}
		cache := &createLimitCacheStub{count: 1000}
		svc := &APIKeyService{apiKeyRepo: repo, cache: cache, settingService: settings}

		require.NoError(t, svc.checkAPIKeyCreateLimits(ctx, 7))
	})

	t.Run("database and cache errors fail open", func(t *testing.T) {
		settings := NewSettingService(&settingRepoStub{values: map[string]string{
			SettingKeyAPIKeyMaxActivePerUser:  "2",
			SettingKeyAPIKeyCreateHourlyLimit: "3",
		}}, nil)
		svc := &APIKeyService{
			apiKeyRepo:     &createLimitRepoStub{err: errors.New("database unavailable")},
			cache:          &createLimitCacheStub{err: errors.New("redis unavailable")},
			settingService: settings,
		}

		require.NoError(t, svc.checkAPIKeyCreateLimits(ctx, 7))
	})
}
