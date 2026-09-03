//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeQPSCache struct {
	count int
	err   error
}

func (f *fakeQPSCache) IncrementKeyQPS(_ context.Context, _ int64) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.count++
	return f.count, nil
}

func TestNormalizeAPIKeyQPSLimit(t *testing.T) {
	require.Equal(t, 0, normalizeAPIKeyQPSLimit(0))
	require.Equal(t, 0, normalizeAPIKeyQPSLimit(-3))
	require.Equal(t, 10, normalizeAPIKeyQPSLimit(10))
	require.Equal(t, MaxAPIKeyQPSLimit, normalizeAPIKeyQPSLimit(MaxAPIKeyQPSLimit+1))
}

func TestCheckAndIncrementKeyQPS(t *testing.T) {
	ctx := context.Background()

	t.Run("nil limit or nil cache allows", func(t *testing.T) {
		svc := &APIKeyService{}
		require.NoError(t, svc.CheckAndIncrementKeyQPS(ctx, &APIKey{ID: 1, QPSLimit: 0}))
		svc.SetQPSCounterCache(nil)
		require.NoError(t, svc.CheckAndIncrementKeyQPS(ctx, &APIKey{ID: 1, QPSLimit: 5}))
	})

	t.Run("over limit rejected", func(t *testing.T) {
		cache := &fakeQPSCache{}
		svc := &APIKeyService{}
		svc.SetQPSCounterCache(cache)
		key := &APIKey{ID: 7, QPSLimit: 2}
		require.NoError(t, svc.CheckAndIncrementKeyQPS(ctx, key))
		require.NoError(t, svc.CheckAndIncrementKeyQPS(ctx, key))
		require.ErrorIs(t, svc.CheckAndIncrementKeyQPS(ctx, key), ErrAPIKeyQPSLimited)
	})

	t.Run("counter failure fails open", func(t *testing.T) {
		cache := &fakeQPSCache{err: errors.New("redis down")}
		svc := &APIKeyService{}
		svc.SetQPSCounterCache(cache)
		require.NoError(t, svc.CheckAndIncrementKeyQPS(ctx, &APIKey{ID: 7, QPSLimit: 1}))
	})
}
