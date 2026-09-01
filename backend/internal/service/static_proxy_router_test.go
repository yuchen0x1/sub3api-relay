//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// routingSettingService builds a SettingService whose static_ip_pool_enabled
// flag resolves from the provided raw settings map.
func routingSettingService(vals map[string]string) *SettingService {
	if vals == nil {
		vals = map[string]string{}
	}
	return NewSettingService(&fakeSettingRepo{vals: vals}, nil)
}

func TestStaticProxyRouterNilSafe(t *testing.T) {
	var nilRouter *StaticProxyRouter
	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	release, err := nilRouter.Apply(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	require.Equal(t, "legacy.example.com", account.Proxy.Host)

	// Missing lookup / concurrency collaborators: legacy path.
	router := NewStaticProxyRouter(nil, nil, nil)
	release, err = router.Apply(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)

	// Nil account: no-op.
	deployed := NewStaticProxyRouter(&staticProxyRoutingLookupFake{}, NewConcurrencyService(&staticProxyConcurrencyCacheFake{}), nil)
	release, err = deployed.Apply(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, release)
}

func TestStaticProxyRouterDisabledByFlagKeepsLegacyPath(t *testing.T) {
	lookup := &staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
		},
	}
	router := NewStaticProxyRouter(lookup, NewConcurrencyService(&staticProxyConcurrencyCacheFake{}), routingSettingService(map[string]string{
		SettingKeyStaticIPPoolEnabled: "false",
	}))
	require.False(t, router.Enabled(context.Background()))

	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	release, err := router.Apply(context.Background(), account)
	require.NoError(t, err)
	require.Nil(t, release)
	// The flag is the canary rollback switch: with it off, the account must
	// keep its legacy proxy and never reserve a per-exit slot.
	require.Equal(t, "legacy.example.com", account.Proxy.Host)
	require.Nil(t, account.ProxyID)
	require.Empty(t, account.StaticProxyIDs)
}

func TestStaticProxyRouterEnabledByDefaultMissingFlag(t *testing.T) {
	// No setting stored → v0.1.198 behavior stays on (fail-open default).
	cache := &staticProxyConcurrencyCacheFake{
		acquire: func(_ context.Context, _ int64, staticProxyID int64, _ int, _ string) (bool, error) {
			return staticProxyID == 11, nil
		},
	}
	router := NewStaticProxyRouter(&staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
		},
	}, NewConcurrencyService(cache), routingSettingService(nil))
	require.True(t, router.Enabled(context.Background()))

	account := &Account{ID: 42, Proxy: testProxy(1, "legacy.example.com")}
	release, err := router.Apply(context.Background(), account)
	require.NoError(t, err)
	require.NotNil(t, release)
	require.Equal(t, []int64{11}, account.StaticProxyIDs)
	require.Equal(t, int64(101), *account.ProxyID)
	require.Equal(t, "exit11.example.com", account.Proxy.Host)
	release()
	require.Len(t, cache.releaseCalls, 1)
}

func TestStaticProxyRouterFailsClosedOnCapacityExhausted(t *testing.T) {
	router := NewStaticProxyRouter(&staticProxyRoutingLookupFake{
		targets: []StaticProxyRoutingTarget{
			{ExitID: 11, ProxyID: 101, Proxy: testProxy(101, "exit11.example.com")},
		},
	}, NewConcurrencyService(&staticProxyConcurrencyCacheFake{
		acquire: func(context.Context, int64, int64, int, string) (bool, error) {
			return false, nil
		},
	}), routingSettingService(nil))
	account := &Account{ID: 42}
	_, err := router.Apply(context.Background(), account)
	require.ErrorIs(t, err, ErrStaticProxyCapacityUnavailable)
}
