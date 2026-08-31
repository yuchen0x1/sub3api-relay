//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// staticProxyRoutingLookupFake is a minimal StaticProxyRoutingLookup test double
// that returns a fixed target set or error.
type staticProxyRoutingLookupFake struct {
	targets []StaticProxyRoutingTarget
	err     error
}

func (f *staticProxyRoutingLookupFake) ListEnabledRoutingTargets(context.Context, int64) ([]StaticProxyRoutingTarget, error) {
	return f.targets, f.err
}

var _ StaticProxyRoutingLookup = (*staticProxyRoutingLookupFake)(nil)

func testProxy(id int64, host string) *Proxy {
	return &Proxy{ID: id, Name: "p", Protocol: "http", Host: host, Port: 8080, Status: StatusActive}
}

// routingGatewayService builds a GatewayService with optional static-proxy
// routing lookup and concurrency backend so tests can exercise
// applyStaticProxyRouting without touching NewGatewayService's full signature.
func routingGatewayService(lookup StaticProxyRoutingLookup, cache ConcurrencyCache) *GatewayService {
	svc := &GatewayService{}
	if lookup != nil {
		svc.SetStaticProxyRoutingLookup(lookup)
	}
	if cache != nil {
		svc.concurrencyService = NewConcurrencyService(cache)
	}
	return svc
}

func TestApplyStaticProxyRoutingLegacyWhenNoLookupOrConcurrency(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	require.Empty(t, account.StaticProxyIDs)
	require.Equal(t, "legacy.example.com", account.Proxy.Host)
}

func TestApplyStaticProxyRoutingLegacyWhenLookupFailsOpen(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{err: errors.New("db down")}
	svc := routingGatewayService(lookup, &staticProxyConcurrencyCacheFake{})
	account := &Account{ID: 42}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	require.Empty(t, account.StaticProxyIDs)
}

func TestApplyStaticProxyRoutingLegacyWhenNoTargets(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{}
	svc := routingGatewayService(lookup, &staticProxyConcurrencyCacheFake{})
	account := &Account{ID: 42}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	require.Empty(t, account.StaticProxyIDs)
}

func TestApplyStaticProxyRoutingFailsOpenOnSlotBackendError(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
		},
	}
	cache := &staticProxyConcurrencyCacheFake{
		acquire: func(context.Context, int64, int64, int, string) (bool, error) {
			return false, errors.New("redis down")
		},
	}
	svc := routingGatewayService(lookup, cache)
	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	// Fail-open must never re-point the account at a static proxy it could not
	// reserve: the legacy proxy remains in place.
	require.Equal(t, "legacy.example.com", account.Proxy.Host)
	require.Nil(t, account.ProxyID)
}

func TestApplyStaticProxyRoutingFailsClosedOnCapacityExhausted(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
		},
	}
	cache := &staticProxyConcurrencyCacheFake{
		acquire: func(context.Context, int64, int64, int, string) (bool, error) {
			return false, nil
		},
	}
	svc := routingGatewayService(lookup, cache)
	account := &Account{ID: 42}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.ErrorIs(t, err, ErrStaticProxyCapacityUnavailable)
	require.Nil(t, release)
	require.True(t, staticProxyCapacityUnavailable(err))
}

func TestApplyStaticProxyRoutingBindsSelectedProxyAndReturnsRelease(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
			{ExitID: 12, ProxyID: 102, Proxy: testProxy(102, "exit12.example.com")},
		},
	}
	cache := &staticProxyConcurrencyCacheFake{
		acquire: func(_ context.Context, _ int64, staticProxyID int64, _ int, _ string) (bool, error) {
			return staticProxyID == 12, nil
		},
	}
	svc := routingGatewayService(lookup, cache)
	account := &Account{ID: 42}
	release, err := svc.applyStaticProxyRouting(context.Background(), account)
	require.NoError(t, err)
	require.NotNil(t, release)
	require.Equal(t, []int64{11, 12}, account.StaticProxyIDs)
	require.Equal(t, int64(102), *account.ProxyID)
	require.Equal(t, "exit12.example.com", account.Proxy.Host)

	// Release must reach the cache release path once and free the slot.
	release()
	require.Len(t, cache.releaseCalls, 1)
	require.Equal(t, int64(42), cache.releaseCalls[0].accountID)
	require.Equal(t, int64(12), cache.releaseCalls[0].staticProxyID)
	require.NotEmpty(t, cache.releaseCalls[0].requestID)

	// The returned release func is idempotent: a second call must not double-release.
	release()
	require.Len(t, cache.releaseCalls, 1)
}

func TestBindStaticProxyForwardTargetMissingProxy(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{ID: 42}
	require.False(t, svc.bindStaticProxyForwardTarget(account, []StaticProxyRoutingTarget{
		{ExitID: 11, ProxyID: 101, Proxy: nil},
	}, 11))
	require.Nil(t, account.Proxy)
	require.Nil(t, account.ProxyID)
}

func TestBindStaticProxyForwardTargetNoMatch(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	require.False(t, svc.bindStaticProxyForwardTarget(account, []StaticProxyRoutingTarget{
		{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
	}, 99))
	// Legacy proxy must remain untouched.
	require.Equal(t, "legacy.example.com", account.Proxy.Host)
}

func TestStaticProxyExitIDs(t *testing.T) {
	require.Empty(t, staticProxyExitIDs(nil))
	require.Equal(t, []int64{11, 12}, staticProxyExitIDs([]StaticProxyRoutingTarget{
		{ExitID: 11}, {ExitID: 12},
	}))
}
