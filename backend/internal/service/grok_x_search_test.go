//go:build unit

package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildGrokXSearchResponsesBody(t *testing.T) {
	inbound := []byte(`{"query":"grok 4.6 release notes","model":"grok-4.6","settings":{"max_results":10}}`)

	body, err := buildGrokXSearchResponsesBody(inbound, "grok-4.6")
	require.NoError(t, err)

	require.Equal(t, "grok-4.6", gjson.GetBytes(body, "model").String())
	require.True(t, gjson.GetBytes(body, "stream").Bool())
	toolType := gjson.GetBytes(body, "tools.0.type").String()
	require.Equal(t, "x_search", toolType)
	prompt := gjson.GetBytes(body, "input.0.content.0.text").String()
	require.Contains(t, prompt, "grok 4.6 release notes")
	require.Contains(t, prompt, "x_search tool")
}

func TestBuildGrokXSearchResponsesBody_DefaultPromptWithoutQuery(t *testing.T) {
	body, err := buildGrokXSearchResponsesBody([]byte(`{}`), "grok-4.5")
	require.NoError(t, err)
	prompt := gjson.GetBytes(body, "input.0.content.0.text").String()
	require.NotEmpty(t, prompt)
}

func TestBuildGrokXSearchResponsesBody_EmptyModel(t *testing.T) {
	_, err := buildGrokXSearchResponsesBody([]byte(`{}`), " ")
	require.Error(t, err)
}

func TestApplyGrokTokenClaims_ExtractsSubscriptionTier(t *testing.T) {
	makeJWT := func(payload map[string]any) string {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		return "header." + base64.RawURLEncoding.EncodeToString(raw) + ".signature"
	}

	t.Run("subscription_tier claim preferred", func(t *testing.T) {
		info := &GrokTokenInfo{}
		applyGrokTokenClaims(info, makeJWT(map[string]any{"subscription_tier": "supergrok_heavy"}))
		require.Equal(t, "supergrok_heavy", info.SubscriptionTier)
	})

	t.Run("falls back to tier claim", func(t *testing.T) {
		info := &GrokTokenInfo{}
		applyGrokTokenClaims(info, makeJWT(map[string]any{"tier": "superGrok"}))
		require.Equal(t, "superGrok", info.SubscriptionTier)
	})

	t.Run("no tier claim leaves empty", func(t *testing.T) {
		info := &GrokTokenInfo{}
		applyGrokTokenClaims(info, makeJWT(map[string]any{"email": "a@b.c"}))
		require.Equal(t, "", info.SubscriptionTier)
	})

	t.Run("garbage token is ignored", func(t *testing.T) {
		info := &GrokTokenInfo{}
		applyGrokTokenClaims(info, "not-a-jwt")
		require.Equal(t, "", info.SubscriptionTier)
	})

	t.Run("existing tier not overwritten", func(t *testing.T) {
		info := &GrokTokenInfo{SubscriptionTier: "free"}
		applyGrokTokenClaims(info, makeJWT(map[string]any{"tier": "superGrok"}))
		require.Equal(t, "free", info.SubscriptionTier)
	})
}

func TestBuildGrokXSearchPrompt_TruncatesLargeInput(t *testing.T) {
	large := strings.Repeat("x", 20000)
	body, err := json.Marshal(map[string]any{"input": large})
	require.NoError(t, err)
	prompt := buildGrokXSearchPrompt(body)
	require.Less(t, len(prompt), 12000)
	require.Contains(t, prompt, "<truncated>")
}
