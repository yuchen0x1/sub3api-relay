//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxReasoningEffortBillingMultiplier(t *testing.T) {
	t.Run("non_max_effort_is_never_multiplied", func(t *testing.T) {
		require.Equal(t, 1.0, maxReasoningEffortBillingMultiplier("gpt-5.6-sol", "high", nil))
		require.Equal(t, 1.0, maxReasoningEffortBillingMultiplier("claude-fable-5-1", "xhigh", nil))
		require.Equal(t, 1.0, maxReasoningEffortBillingMultiplier("gpt-5.6-sol", "", nil))
	})

	t.Run("channel_multiplier_wins_when_configured", func(t *testing.T) {
		pricing := &ModelPricing{MaxReasoningEffortMultiplier: testPtrFloat64(2.5)}
		require.Equal(t, 2.5, maxReasoningEffortBillingMultiplier("gpt-5.6-sol", "max", pricing))
	})

	t.Run("fable_default_applies_without_channel_config", func(t *testing.T) {
		require.Equal(t, 3.0, maxReasoningEffortBillingMultiplier("claude-fable-5-1", "max", nil))
		require.Equal(t, 3.0, maxReasoningEffortBillingMultiplier("Claude-Fable-5.1-Max", "MAX", &ModelPricing{}))
	})

	t.Run("non_fable_without_channel_config_stays_one", func(t *testing.T) {
		require.Equal(t, 1.0, maxReasoningEffortBillingMultiplier("gpt-5.6-sol", "max", nil))
	})

	t.Run("non_positive_channel_multiplier_is_ignored", func(t *testing.T) {
		require.Equal(t, 1.0, maxReasoningEffortBillingMultiplier("gpt-5.6-sol", "max", &ModelPricing{MaxReasoningEffortMultiplier: testPtrFloat64(0)}))
	})
}

func TestIsClaudeFable51Model(t *testing.T) {
	require.True(t, isClaudeFable51Model("claude-fable-5-1"))
	require.True(t, isClaudeFable51Model("claude-fable-5.1"))
	require.True(t, isClaudeFable51Model("claude-fable-5.1-20260101"))
	require.False(t, isClaudeFable51Model("claude-fable-5.2"))
	require.False(t, isClaudeFable51Model("claude-sonnet-4"))
}

func TestCalculateCostUnified_MaxReasoningEffortMultiplierNoResolver(t *testing.T) {
	// 本地 fallback 无 fable 专属价，claude-* 未知型号统一回退 sonnet-4 单价；
	// 倍率仍按请求模型字符串判定（fable → 3x）。
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}

	base, err := svc.CalculateCostUnified(CostInput{Model: "claude-fable-5-1", Tokens: tokens, RateMultiplier: 1})
	require.NoError(t, err)
	require.Positive(t, base.TotalCost)

	multiplied, err := svc.CalculateCostUnified(CostInput{
		Model: "claude-fable-5-1", Tokens: tokens, RateMultiplier: 1, ReasoningEffort: "max",
	})
	require.NoError(t, err)
	require.InDelta(t, base.TotalCost*3, multiplied.TotalCost, base.TotalCost*3*1e-9)
	require.InDelta(t, base.InputCost*3, multiplied.InputCost, base.InputCost*3*1e-9)
	require.InDelta(t, base.OutputCost*3, multiplied.OutputCost, base.OutputCost*3*1e-9)
}

func TestCalculateCostUnified_NonMaxEffortKeepsBaseCost(t *testing.T) {
	svc := newTestBillingServiceWithPrices(map[string]*ModelPricing{
		"claude-fable-5-1": {InputPricePerToken: 1e-6, OutputPricePerToken: 2e-6},
	})
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}

	base, err := svc.CalculateCostUnified(CostInput{Model: "claude-fable-5-1", Tokens: tokens, RateMultiplier: 1})
	require.NoError(t, err)

	high, err := svc.CalculateCostUnified(CostInput{
		Model: "claude-fable-5-1", Tokens: tokens, RateMultiplier: 1, ReasoningEffort: "xhigh",
	})
	require.NoError(t, err)
	require.InDelta(t, base.TotalCost, high.TotalCost, 1e-12)
}

func TestGetModelPricingWithChannel_MaxReasoningEffortMultiplierOverlay(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricingWithChannel("claude-sonnet-4", &ChannelModelPricing{
		MaxReasoningEffortMultiplier: testPtrFloat64(2.5),
	})
	require.NoError(t, err)
	require.NotNil(t, pricing.MaxReasoningEffortMultiplier)
	require.InDelta(t, 2.5, *pricing.MaxReasoningEffortMultiplier, 1e-12)

	// 未配置倍率时不覆盖模型默认值（fable 通过默认规则兜底，其他模型保持 nil）。
	plain, err := svc.GetModelPricingWithChannel("claude-sonnet-4", &ChannelModelPricing{
		InputPrice: testPtrFloat64(4e-6),
	})
	require.NoError(t, err)
	require.Nil(t, plain.MaxReasoningEffortMultiplier)
}

func TestApplyTokenOverridesCarriesMaxReasoningEffortMultiplier(t *testing.T) {
	chPricing := &ChannelModelPricing{MaxReasoningEffortMultiplier: testPtrFloat64(2.0)}

	flat := &ResolvedPricing{BasePricing: &ModelPricing{}}
	r := &ModelPricingResolver{}
	r.applyTokenOverrides(chPricing, flat)
	require.NotNil(t, flat.BasePricing.MaxReasoningEffortMultiplier)
	require.InDelta(t, 2.0, *flat.BasePricing.MaxReasoningEffortMultiplier, 1e-12)

	resolved := &ResolvedPricing{
		BasePricing: &ModelPricing{},
		Intervals: []PricingInterval{
			{MinTokens: 0, InputPrice: testPtrFloat64(1e-6), OutputPrice: testPtrFloat64(2e-6)},
		},
	}
	r.applyTokenOverrides(chPricing, resolved)
	intervalPricing := intervalToModelPricing(&resolved.Intervals[0], false, chPricing)
	require.NotNil(t, intervalPricing.MaxReasoningEffortMultiplier)
	require.InDelta(t, 2.0, *intervalPricing.MaxReasoningEffortMultiplier, 1e-12)
}

func TestTryModelFilePricing_AppliesMaxReasoningEffortMultiplier(t *testing.T) {
	// fable 未知型号回退 sonnet-4 单价：input 3e-6、output 15e-6。
	bs := newTestBillingService()
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50}

	result := tryModelFilePricing(bs, "claude-fable-5-1", tokens, "")
	require.InDelta(t, 100*3e-6+50*15e-6, *result, 1e-12)

	maxed := tryModelFilePricing(bs, "claude-fable-5-1", tokens, "max")
	require.InDelta(t, (100*3e-6+50*15e-6)*3, *maxed, 1e-12)
}
