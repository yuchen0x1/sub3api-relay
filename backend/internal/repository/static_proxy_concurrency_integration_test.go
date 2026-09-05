//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestStaticProxyConcurrency_EnforcesTenPerAssignmentAndAggregateCapacity(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cache := NewConcurrencyCache(client, 15, 900)
	staticCache, ok := cache.(service.StaticProxyConcurrencyCache)
	require.True(t, ok)
	liveCache, ok := cache.(service.StaticProxyLiveConcurrencyCache)
	require.True(t, ok)
	ctx := context.Background()

	for i := 0; i < service.StaticProxyCapacity; i++ {
		acquired, err := staticCache.AcquireStaticProxySlot(ctx, 42, 101, service.StaticProxyCapacity*2, "regular-"+string(rune('a'+i)))
		require.NoError(t, err)
		require.True(t, acquired)
	}
	acquired, err := staticCache.AcquireStaticProxySlot(ctx, 42, 101, service.StaticProxyCapacity*2, "regular-overflow")
	require.NoError(t, err)
	require.False(t, acquired)

	count, err := staticCache.GetStaticProxyConcurrency(ctx, 42, 101)
	require.NoError(t, err)
	require.Equal(t, service.StaticProxyCapacity, count)

	for i := 0; i < service.StaticProxyCapacity; i++ {
		acquired, err := staticCache.AcquireStaticProxySlot(ctx, 42, 102, service.StaticProxyCapacity*2, "second-"+string(rune('a'+i)))
		require.NoError(t, err)
		require.True(t, acquired)
	}
	acquired, err = liveCache.AcquireStaticProxyLiveSlot(ctx, 42, 102, service.StaticProxyCapacity*2, "live-overflow")
	require.NoError(t, err)
	require.False(t, acquired, "aggregate capacity must include regular reservations")

	require.NoError(t, staticCache.ReleaseStaticProxySlot(ctx, 42, 101, "regular-a"))
	require.NoError(t, staticCache.ReleaseStaticProxySlot(ctx, 42, 101, "regular-a"))
	acquired, err = liveCache.AcquireStaticProxyLiveSlot(ctx, 42, 101, service.StaticProxyCapacity*2, "live-reused")
	require.NoError(t, err)
	require.True(t, acquired)
	count, err = staticCache.GetStaticProxyConcurrency(ctx, 42, 101)
	require.NoError(t, err)
	require.Equal(t, service.StaticProxyCapacity, count)
	require.NoError(t, liveCache.ReleaseStaticProxyLiveSlot(ctx, 42, 101, "live-reused"))
}
