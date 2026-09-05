//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/claude"
)

func TestStripDeferredToolCacheControl(t *testing.T) {
	body := []byte(`{"tools":[
		{"name":"a","defer_loading":true,"cache_control":{"type":"ephemeral"}},
		{"name":"b","custom":{"defer_loading":true},"cache_control":{"type":"ephemeral","ttl":"1h"}},
		{"name":"c","cache_control":{"type":"ephemeral"}},
		{"name":"d","defer_loading":false,"cache_control":{"type":"ephemeral"}}
	]}`)

	stripped := stripDeferredToolCacheControl(body)
	tools := gjson.GetBytes(stripped, "tools").Array()
	require.Len(t, tools, 4)
	require.False(t, tools[0].Get("cache_control").Exists(), "deferred tool loses cache_control")
	require.False(t, tools[1].Get("cache_control").Exists(), "custom.defer_loading also stripped")
	require.True(t, tools[2].Get("cache_control").Exists(), "regular tool keeps cache_control")
	require.True(t, tools[3].Get("cache_control").Exists(), "explicit false is not deferred")

	require.Equal(t, stripped, stripDeferredToolCacheControl(stripped), "idempotent")
}

func TestApplyToolsLastCacheBreakpointSkipsDeferredTools(t *testing.T) {
	body := []byte(`{"tools":[
		{"name":"a","defer_loading":true},
		{"name":"b"}
	]}`)

	out := applyToolsLastCacheBreakpoint(body)
	tools := gjson.GetBytes(out, "tools").Array()
	require.True(t, tools[1].Get("cache_control").Exists(), "breakpoint lands on last non-deferred tool")
	require.False(t, tools[0].Get("cache_control").Exists(), "deferred tool stays clean")

	allDeferred := []byte(`{"tools":[{"name":"a","defer_loading":true}]}`)
	require.Equal(t, allDeferred, applyToolsLastCacheBreakpoint(allDeferred), "no injectable target: no-op")
}

func TestEffectiveBillingUserAgent(t *testing.T) {
	fp := &Fingerprint{UserAgent: "claude-cli/2.0.0 (cached)"}
	builtin := claude.DefaultHeaders["User-Agent"]

	require.Equal(t, builtin, effectiveBillingUserAgent("oauth", true, fp), "mimicry forces builtin UA")
	require.Equal(t, fp.UserAgent, effectiveBillingUserAgent("oauth", false, fp), "no mimicry keeps fingerprint")
	require.Equal(t, builtin, effectiveBillingUserAgent("oauth", true, nil))
	require.Equal(t, "", effectiveBillingUserAgent("oauth", false, nil), "no fingerprint and no mimicry: no sync")
}
