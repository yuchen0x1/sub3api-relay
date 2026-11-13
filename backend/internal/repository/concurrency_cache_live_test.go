package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestLiveLeaseReplacesRegularSlotsAndCountsTowardLimits(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	regular := NewConcurrencyCache(client, 15, 900)
	live, ok := regular.(service.LiveConcurrencyCache)
	require.True(t, ok)
	ctx := context.Background()

	accountAcquired, err := regular.AcquireAccountSlot(ctx, 10, 1, "regular-account")
	require.NoError(t, err)
	require.True(t, accountAcquired)
	userAcquired, err := regular.AcquireUserSlot(ctx, 20, 1, "regular-user")
	require.NoError(t, err)
	require.True(t, userAcquired)

	acquired, err := live.AcquireLiveLease(ctx, 10, 1, 20, 1, 30, "live-lease", true)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, regular.ReleaseAccountSlot(ctx, 10, "regular-account"))
	require.NoError(t, regular.ReleaseUserSlot(ctx, 20, "regular-user"))

	accountCount, err := regular.GetAccountConcurrency(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, accountCount)
	userCount, err := regular.GetUserConcurrency(ctx, 20)
	require.NoError(t, err)
	require.Equal(t, 1, userCount)
	accountAcquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-blocked")
	require.NoError(t, err)
	require.False(t, accountAcquired)

	refreshed, err := live.RefreshLiveLease(ctx, 10, 20, 30, "live-lease")
	require.NoError(t, err)
	require.True(t, refreshed)
	require.NoError(t, live.ReleaseLiveLease(ctx, 10, 20, 30, "live-lease"))
	accountAcquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-allowed")
	require.NoError(t, err)
	require.True(t, accountAcquired)
}

func TestLiveLeaseExpiresWithoutRefresh(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	regular := NewConcurrencyCache(client, 15, 900)
	live, ok := regular.(service.LiveConcurrencyCache)
	require.True(t, ok)
	ctx := context.Background()

	acquired, err := live.AcquireLiveLease(ctx, 10, 1, 20, 1, 30, "expired-live", false)
	require.NoError(t, err)
	require.True(t, acquired)

	redisServer.FastForward(61 * time.Second)
	acquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-after-expiry")
	require.NoError(t, err)
	require.True(t, acquired)
	refreshed, err := live.RefreshLiveLease(ctx, 10, 20, 30, "expired-live")
	require.NoError(t, err)
	require.False(t, refreshed)
}

// staticProxyLiveLeaseFixture 构造一个持有 live static-proxy 槽位的缓存。
func staticProxyLiveLeaseFixture(t *testing.T) (*concurrencyCache, *miniredis.Miniredis, service.StaticProxyLiveConcurrencyCache, context.Context) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache, ok := NewConcurrencyCache(client, 15, 900).(*concurrencyCache)
	require.True(t, ok)
	liveCache, ok := any(cache).(service.StaticProxyLiveConcurrencyCache)
	require.True(t, ok, "concurrencyCache must implement StaticProxyLiveConcurrencyCache")
	return cache, redisServer, liveCache, context.Background()
}

// RefreshStaticProxyLiveSlot 是可续期能力：租约到期前续期必须成功，
// 续期后计数保持不变，且未持有的 requestID 不能凭空续期。
func TestStaticProxyLiveSlotRefresh(t *testing.T) {
	cache, redisServer, liveCache, ctx := staticProxyLiveLeaseFixture(t)
	const accountID, staticProxyID = int64(42), int64(101)

	acquired, err := liveCache.AcquireStaticProxyLiveSlot(ctx, accountID, staticProxyID, service.StaticProxyCapacity*2, "live-refresh")
	require.NoError(t, err)
	require.True(t, acquired)

	refreshed, err := liveCache.RefreshStaticProxyLiveSlot(ctx, accountID, staticProxyID, "live-refresh")
	require.NoError(t, err)
	require.True(t, refreshed)

	// 续期不应重复计数，也不应改变聚合视图。
	count, err := liveCache.GetStaticProxyLiveConcurrency(ctx, accountID, staticProxyID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	aggregate, err := cache.GetStaticProxyConcurrency(ctx, accountID, staticProxyID)
	require.NoError(t, err)
	require.Equal(t, 1, aggregate)

	// 未持有的 requestID 续期必须失败，避免越权延寿。
	refreshed, err = liveCache.RefreshStaticProxyLiveSlot(ctx, accountID, staticProxyID, "not-holder")
	require.NoError(t, err)
	require.False(t, refreshed)

	// 续期后租约已延长：越过一个 TTL 仍然存活。
	redisServer.FastForward(liveLeaseTTLSeconds/2 + 1)
	count, err = liveCache.GetStaticProxyLiveConcurrency(ctx, accountID, staticProxyID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// 未续期时租约到期后必须失效，且续期接口返回 false（调用方需重新获取）。
func TestStaticProxyLiveSlotRefreshFailsAfterExpiry(t *testing.T) {
	_, redisServer, liveCache, ctx := staticProxyLiveLeaseFixture(t)
	const accountID, staticProxyID = int64(42), int64(102)

	acquired, err := liveCache.AcquireStaticProxyLiveSlot(ctx, accountID, staticProxyID, service.StaticProxyCapacity*2, "live-expiring")
	require.NoError(t, err)
	require.True(t, acquired)

	redisServer.FastForward(time.Duration(liveLeaseTTLSeconds+1) * time.Second)

	refreshed, err := liveCache.RefreshStaticProxyLiveSlot(ctx, accountID, staticProxyID, "live-expiring")
	require.NoError(t, err)
	require.False(t, refreshed)

	count, err := liveCache.GetStaticProxyLiveConcurrency(ctx, accountID, staticProxyID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// 非法入参必须安全返回 false 且不报错（与其它槽位方法的约定一致）。
func TestStaticProxyLiveSlotRefreshRejectsInvalidInput(t *testing.T) {
	_, _, liveCache, ctx := staticProxyLiveLeaseFixture(t)

	for _, tc := range []struct {
		name        string
		accountID   int64
		staticProxy int64
		requestID   string
	}{
		{"zero account", 0, 101, "id"},
		{"negative account", -1, 101, "id"},
		{"zero proxy", 42, 0, "id"},
		{"negative proxy", 42, -5, "id"},
		{"empty request", 42, 101, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refreshed, err := liveCache.RefreshStaticProxyLiveSlot(ctx, tc.accountID, tc.staticProxy, tc.requestID)
			require.NoError(t, err)
			require.False(t, refreshed)
		})
	}
}
