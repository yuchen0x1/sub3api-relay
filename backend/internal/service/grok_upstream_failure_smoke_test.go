//go:build unit

package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyGrokUpstreamFailure(t *testing.T) {
	t.Run("free usage scoped to a model", func(t *testing.T) {
		d := classifyGrokUpstreamFailure(http.StatusTooManyRequests,
			[]byte(`{"error":{"message":"You have used all the free usage for model grok-4.5"}}`), "grok-4.5")
		require.Equal(t, GrokFailureFreeUsage, d.Class)
		require.True(t, d.ShouldCooldown)
		require.True(t, d.ShouldFailover)
		require.Equal(t, "grok-4.5", d.Model)
	})

	t.Run("payment required without quota language", func(t *testing.T) {
		d := classifyGrokUpstreamFailure(http.StatusPaymentRequired, nil, "")
		require.Equal(t, GrokFailureBilling, d.Class)
		require.True(t, d.ShouldCooldown)
	})

	t.Run("client validation errors do not cool", func(t *testing.T) {
		d := classifyGrokUpstreamFailure(http.StatusBadRequest, []byte(`{"error":{"message":"field x is required"}}`), "")
		require.False(t, d.ShouldCooldown)
	})
}

func TestGrokModelQuotaBlockLifecycle(t *testing.T) {
	now := time.Now()
	accounts := []Account{{ID: 42}, {ID: 44}}
	blockedModel := canonicalOpenAIAccountSchedulingModel(&accounts[0], "grok-4.5")

	markGrokModelQuotaBlock(42, blockedModel, now.Add(30*time.Minute))
	require.True(t, isGrokModelQuotaBlocked(42, blockedModel, now))
	require.False(t, isGrokModelQuotaBlocked(43, blockedModel, now), "other accounts unaffected")
	require.False(t, isGrokModelQuotaBlocked(42, "grok-5", now), "other models unaffected")

	filtered := filterGrokModelQuotaBlockedAccounts(accounts, blockedModel, now)
	require.Len(t, filtered, 1)
	require.Equal(t, int64(44), filtered[0].ID)

	// 过期读取会顺带从存储删除条目，放在最后断言。
	require.False(t, isGrokModelQuotaBlocked(42, blockedModel, now.Add(time.Hour)), "expired block released")
}

func TestGrokTeamRateLimitContextHelpersAreSafe(t *testing.T) {
	require.Nil(t, withGrokTeamRateLimitModel(nil, ""))
	require.Equal(t, "", grokRequestedModelFromCtx(nil))
}
