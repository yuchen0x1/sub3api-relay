//go:build unit

package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func newOverrideTestService(t *testing.T, overrideJSON string) *PricingService {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "override.json")
	require.NoError(t, os.WriteFile(path, []byte(overrideJSON), 0644))
	cfg := &config.Config{}
	cfg.Pricing.OverrideFile = path
	return &PricingService{cfg: cfg}
}

func TestApplyPricingOverrides(t *testing.T) {
	svc := newOverrideTestService(t, `{
		"claude-sonnet-4": {"input_cost_per_token": 9e-6, "cache_read_input_token_cost": null},
		"gpt-brand-new":   {"input_cost_per_token": 2e-6, "output_cost_per_token": 8e-6}
	}`)

	raw := map[string]json.RawMessage{
		"claude-sonnet-4": json.RawMessage(`{"input_cost_per_token":3e-6,"output_cost_per_token":15e-6,"cache_read_input_token_cost":0.3e-6,"litellm_provider":"anthropic"}`),
	}
	out := svc.applyPricingOverrides(raw)

	body, err := json.Marshal(out)
	require.NoError(t, err)
	parsed, err := svc.parsePricingData(body)
	require.NoError(t, err)

	entry, ok := parsed["claude-sonnet-4"]
	require.True(t, ok, "existing model must survive override merge")
	require.InDelta(t, 9e-6, entry.InputCostPerToken, 1e-15, "patched field wins")
	require.InDelta(t, 15e-6, entry.OutputCostPerToken, 1e-15, "untouched field preserved")
	require.Zero(t, entry.CacheReadInputTokenCost, "null patch field deletes the key")

	// override-only 模型走 mergeOverrideOnlyModels 并入
	merged := svc.mergeOverrideOnlyModels(parsed)
	newEntry, ok := merged["gpt-brand-new"]
	require.True(t, ok, "override-only model with price fields must be added")
	require.InDelta(t, 2e-6, newEntry.InputCostPerToken, 1e-15)
}

func TestValidateCustomPricingFiles(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte("not-json"), 0644))

	cfg := &config.Config{}
	cfg.Pricing.OverrideFile = bad
	svc := &PricingService{cfg: cfg}
	require.Error(t, svc.validateCustomPricingFiles())

	// 文件不存在是合法状态
	missing := filepath.Join(dir, "missing.json")
	cfg2 := &config.Config{}
	cfg2.Pricing.OverrideFile = missing
	require.NoError(t, (&PricingService{cfg: cfg2}).validateCustomPricingFiles())
}
