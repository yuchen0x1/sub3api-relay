//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func TestNormalizeAccountWeight(t *testing.T) {
	require.Equal(t, 1, normalizeAccountWeight(0))
	require.Equal(t, 1, normalizeAccountWeight(-5))
	require.Equal(t, 1, normalizeAccountWeight(1))
	require.Equal(t, 42, normalizeAccountWeight(42))
	require.Equal(t, MaxAccountWeight, normalizeAccountWeight(MaxAccountWeight+1))
}

func TestNormalizeSchedulingSelectionMode(t *testing.T) {
	require.Equal(t, "", normalizeSchedulingSelectionMode(""))
	require.Equal(t, "", normalizeSchedulingSelectionMode("bogus"))
	require.Equal(t, SchedulingSelectionModeLastUsed, normalizeSchedulingSelectionMode("last_used"))
	require.Equal(t, SchedulingSelectionModeRandom, normalizeSchedulingSelectionMode(" RANDOM "))
	require.Equal(t, SchedulingSelectionModeWeighted, normalizeSchedulingSelectionMode("Weighted"))
}

func TestSelectByWeightedRandom(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		require.Nil(t, selectByWeightedRandom(nil))
	})

	t.Run("single account", func(t *testing.T) {
		accounts := []accountWithLoad{{account: &Account{ID: 1}, loadInfo: &AccountLoadInfo{}}}
		require.NotNil(t, selectByWeightedRandom(accounts))
	})

	t.Run("extreme weight dominates", func(t *testing.T) {
		accounts := []accountWithLoad{
			{account: &Account{ID: 1, Weight: 1}, loadInfo: &AccountLoadInfo{}},
			{account: &Account{ID: 2, Weight: MaxAccountWeight}, loadInfo: &AccountLoadInfo{}},
		}
		for i := 0; i < 50; i++ {
			require.Equal(t, int64(2), selectByWeightedRandom(accounts).account.ID)
		}
	})

	t.Run("equal weights keep both selectable", func(t *testing.T) {
		accounts := []accountWithLoad{
			{account: &Account{ID: 1, Weight: 0}, loadInfo: &AccountLoadInfo{}}, // 0 归一为 1
			{account: &Account{ID: 2, Weight: 1}, loadInfo: &AccountLoadInfo{}},
		}
		seen := map[int64]bool{}
		for i := 0; i < 200; i++ {
			seen[selectByWeightedRandom(accounts).account.ID] = true
		}
		require.Len(t, seen, 2)
	})
}

func TestWeightedShuffleAccounts(t *testing.T) {
	t.Run("keeps permutation", func(t *testing.T) {
		accounts := []*Account{
			{ID: 1, Weight: 1},
			{ID: 2, Weight: 100},
			{ID: 3, Weight: 1},
			{ID: 4, Weight: 1},
		}
		orig := map[int64]bool{}
		for _, a := range accounts {
			orig[a.ID] = true
		}
		for i := 0; i < 20; i++ {
			shuffled := append([]*Account(nil), accounts...)
			weightedShuffleAccounts(shuffled)
			seen := map[int64]bool{}
			for _, a := range shuffled {
				seen[a.ID] = true
			}
			require.Equal(t, orig, seen)
		}
	})

	t.Run("heavy account usually placed earlier", func(t *testing.T) {
		accounts := []*Account{
			{ID: 1, Weight: 1},
			{ID: 2, Weight: MaxAccountWeight},
			{ID: 3, Weight: 1},
		}
		earlyCount := 0
		trials := 100
		for i := 0; i < trials; i++ {
			shuffled := append([]*Account(nil), accounts...)
			weightedShuffleAccounts(shuffled)
			if shuffled[0].ID == 2 {
				earlyCount++
			}
		}
		// weight 100/102 ≈ 98% 的理论首位的概率，留出随机波动空间
		require.Greater(t, earlyCount, trials*9/10)
	})
}

func TestWeightedShuffleWithinPriority(t *testing.T) {
	accounts := []*Account{
		{ID: 1, Priority: 1, Weight: MaxAccountWeight},
		{ID: 2, Priority: 2, Weight: 1},
		{ID: 3, Priority: 2, Weight: 1},
		{ID: 4, Priority: 3, Weight: 1},
	}
	weightedShuffleWithinPriority(accounts)
	// 打乱后同优先级仍必须连续，且优先级保持升序（段间不交叉）
	require.Equal(t, int64(1), accounts[0].ID)
	require.Equal(t, int64(4), accounts[3].ID)
	require.Equal(t, 2, accounts[1].Priority)
	require.Equal(t, 2, accounts[2].Priority)
}

func TestEffectiveSchedulingSelectionMode(t *testing.T) {
	gs := &GatewayService{}

	t.Run("defaults to last_used", func(t *testing.T) {
		require.Equal(t, SchedulingSelectionModeLastUsed, gs.effectiveSchedulingSelectionMode(context.Background(), config.GatewaySchedulingConfig{}))
	})

	t.Run("config random and weighted pass through", func(t *testing.T) {
		require.Equal(t, SchedulingSelectionModeRandom, gs.effectiveSchedulingSelectionMode(context.Background(), config.GatewaySchedulingConfig{FallbackSelectionMode: "random"}))
		require.Equal(t, SchedulingSelectionModeWeighted, gs.effectiveSchedulingSelectionMode(context.Background(), config.GatewaySchedulingConfig{FallbackSelectionMode: "weighted"}))
	})

	t.Run("invalid config falls back", func(t *testing.T) {
		require.Equal(t, SchedulingSelectionModeLastUsed, gs.effectiveSchedulingSelectionMode(context.Background(), config.GatewaySchedulingConfig{FallbackSelectionMode: "bogus"}))
	})

	t.Run("setting overrides config", func(t *testing.T) {
		repo := newMockSettingRepo()
		require.NoError(t, repo.Set(context.Background(), SettingKeyGatewaySchedulingSelectionMode, "random"))
		svc := NewSettingService(repo, &config.Config{})
		gs := &GatewayService{settingService: svc}

		require.Equal(t, SchedulingSelectionModeRandom, gs.effectiveSchedulingSelectionMode(context.Background(), config.GatewaySchedulingConfig{FallbackSelectionMode: SchedulingSelectionModeLastUsed}))
	})
}

func TestSelectLayeredCandidateWeighted(t *testing.T) {
	accounts := []accountWithLoad{
		{account: &Account{ID: 1, Weight: 1}, loadInfo: &AccountLoadInfo{}},
		{account: &Account{ID: 2, Weight: MaxAccountWeight}, loadInfo: &AccountLoadInfo{}},
	}
	gs := &GatewayService{}
	for i := 0; i < 20; i++ {
		selected := gs.selectLayeredCandidate(accounts, false, SchedulingSelectionModeWeighted)
		require.Equal(t, int64(2), selected.account.ID)
	}
	// 非 weighted 模式不改变 LRU 语义：nil LastUsedAt 的平局随机，但必须是候选之一
	selected := gs.selectLayeredCandidate(accounts, false, SchedulingSelectionModeLastUsed)
	require.Contains(t, []int64{1, 2}, selected.account.ID)
}

func TestBuildAccountForCreateWeight(t *testing.T) {
	input := &CreateAccountInput{
		Name:     "acc",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Weight:   0, // 0 视为默认 1
	}
	acc, err := buildAccountForCreate(input, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, 1, acc.Weight)

	input.Weight = 7
	acc, err = buildAccountForCreate(input, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, 7, acc.Weight)
}
