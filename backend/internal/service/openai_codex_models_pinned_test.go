//go:build unit

package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCodexModelsManifestConfig(t *testing.T) {
	t.Run("non_openai_platform_normalizes_to_zero", func(t *testing.T) {
		got := normalizeCodexModelsManifestConfig(PlatformAnthropic, GroupCodexModelsManifestConfig{
			Enabled: true, AccountIDs: []int64{1, 2}, FallbackToScheduler: true,
		})
		require.Equal(t, GroupCodexModelsManifestConfig{}, got)
	})

	t.Run("dedupes_and_drops_invalid_ids_preserving_order", func(t *testing.T) {
		got := normalizeCodexModelsManifestConfig(PlatformOpenAI, GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{7, 3, 7, 0, -1, 3, 9},
		})
		require.Equal(t, []int64{7, 3, 9}, got.AccountIDs)
		require.True(t, got.Enabled)
	})

	t.Run("disabled_config_keeps_account_list", func(t *testing.T) {
		got := normalizeCodexModelsManifestConfig(PlatformOpenAI, GroupCodexModelsManifestConfig{
			AccountIDs: []int64{5},
		})
		require.False(t, got.Enabled)
		require.Equal(t, []int64{5}, got.AccountIDs)
	})

	t.Run("all_invalid_ids_collapse_to_nil", func(t *testing.T) {
		got := normalizeCodexModelsManifestConfig(PlatformOpenAI, GroupCodexModelsManifestConfig{
			Enabled:    true,
			AccountIDs: []int64{0, -3},
		})
		require.Nil(t, got.AccountIDs)
	})
}

func TestIsPinnedCodexModelsAccountUsable(t *testing.T) {
	require.True(t, isPinnedCodexModelsAccountUsable(&Account{Status: StatusActive, Schedulable: true}))
	require.False(t, isPinnedCodexModelsAccountUsable(&Account{Schedulable: true}))
	require.False(t, isPinnedCodexModelsAccountUsable(&Account{Status: StatusActive, Schedulable: false}))
	require.False(t, isPinnedCodexModelsAccountUsable(nil))
}

func TestMergeCodexModelsManifestBodies(t *testing.T) {
	first := []byte(`{"objects":{"object":"list"},"models":[{"slug":"gpt-5.6"},{"slug":"gpt-5.4"}]}`)
	second := []byte(`{"objects":{"object":"list"},"models":[{"slug":"gpt-5.6"},{"slug":"gpt-5.5"}]}`)

	merged, err := mergeCodexModelsManifestBodies([][]byte{first, second})
	require.NoError(t, err)
	require.JSONEq(t, `{"objects":{"object":"list"},"models":[{"slug":"gpt-5.6"},{"slug":"gpt-5.4"},{"slug":"gpt-5.5"}]}`, string(merged))
}

func TestMergeCodexModelsManifestBodies_RejectsInvalidInput(t *testing.T) {
	_, err := mergeCodexModelsManifestBodies(nil)
	require.Error(t, err)

	_, err = mergeCodexModelsManifestBodies([][]byte{[]byte("not-json")})
	require.Error(t, err)
}

func TestErrNoPinnedCodexModelsAccounts_IsDistinct(t *testing.T) {
	require.True(t, errors.Is(ErrNoPinnedCodexModelsAccounts, ErrNoPinnedCodexModelsAccounts))
	require.False(t, errors.Is(ErrNoPinnedCodexModelsAccounts, errors.New("other")))
}
