package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func TestNewDashboardCacheKeyPrefix(t *testing.T) {
	cache := NewDashboardCache(nil, &config.Config{
		Dashboard: config.DashboardCacheConfig{
			KeyPrefix: "prod",
		},
	})
	impl, ok := cache.(*dashboardCache)
	require.True(t, ok)
	require.Equal(t, "prod:", impl.keyPrefix)

	cache = NewDashboardCache(nil, &config.Config{
		Dashboard: config.DashboardCacheConfig{
			KeyPrefix: "staging:",
		},
	})
	impl, ok = cache.(*dashboardCache)
	require.True(t, ok)
	require.Equal(t, "staging:", impl.keyPrefix)
}
