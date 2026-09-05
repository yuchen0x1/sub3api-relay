//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsOpenAIGPT6AstraModel(t *testing.T) {
	valid := []string{"gpt-6", "GPT-6", "gpt-6-astra", "gpt-6-astra-20260101", "prefix/gpt-6-astra"}
	for _, m := range valid {
		require.Truef(t, isOpenAIGPT6AstraModel(m), "model %q should be Astra", m)
	}
	invalid := []string{"gpt-6-codex", "gpt-6o", "gpt-5.6", "gpt-5"}
	for _, m := range invalid {
		require.Falsef(t, isOpenAIGPT6AstraModel(m), "model %q should not be Astra", m)
	}
}

func TestNormalizeKnownOpenAICodexModelAstra(t *testing.T) {
	require.Equal(t, "gpt-6-astra", normalizeKnownOpenAICodexModel("gpt-6"))
	require.Equal(t, "gpt-6-astra", normalizeKnownOpenAICodexModel("gpt-6-astra"))
}

func TestBillingFallbackPricingForGPT6Astra(t *testing.T) {
	svc := newTestBillingService()
	pricing, err := svc.GetModelPricing("gpt-6")
	require.NoError(t, err)
	require.InDelta(t, 10e-6, pricing.InputPricePerToken, 1e-15)
	require.InDelta(t, 50e-6, pricing.OutputPricePerToken, 1e-15)
	require.InDelta(t, 20e-6, pricing.InputPricePerTokenPriority, 1e-15)

	pricing, err = svc.GetModelPricing("gpt-6-astra")
	require.NoError(t, err)
	require.InDelta(t, 1e-6, pricing.CacheReadPricePerToken, 1e-15)
}
