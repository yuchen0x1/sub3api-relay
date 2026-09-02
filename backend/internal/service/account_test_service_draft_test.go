//go:build unit

package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTestDraftConnectionSurfacesUpstream403Body(t *testing.T) {
	upstream := &queuedHTTPUpstream{responses: []*http.Response{
		newJSONResponse(http.StatusForbidden,
			"{\"error\":{\"message\":\"该令牌不允许在此 IP 使用 (ip: 1.2.3.4)\",\"type\":\"new_api_error\"}}"),
	}}
	svc := &AccountTestService{httpUpstream: upstream}

	result, err := svc.TestDraftConnection(context.Background(), "openai", "apikey", map[string]any{
		"api_key":  "sk-test",
		"base_url": "https://relay.example.com/v1",
	})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "upstream rejected the API key")
	require.Contains(t, result.Error, "HTTP 403")
	require.Contains(t, result.Error, "该令牌不允许在此 IP 使用")
}

func TestTestDraftConnectionSurfacesUpstream401Body(t *testing.T) {
	upstream := &queuedHTTPUpstream{responses: []*http.Response{
		newJSONResponse(http.StatusUnauthorized,
			"{\"error\":{\"message\":\"Invalid token (request id: 20260902)\",\"type\":\"new_api_error\"}}"),
	}}
	svc := &AccountTestService{httpUpstream: upstream}

	result, err := svc.TestDraftConnection(context.Background(), "openai", "apikey", map[string]any{
		"api_key":  "sk-test",
		"base_url": "",
	})
	require.NoError(t, err)
	require.Contains(t, result.Error, "HTTP 401")
	require.Contains(t, result.Error, "Invalid token")
}

func TestDiagnosticBodySnippet(t *testing.T) {
	require.Equal(t, "empty body", diagnosticBodySnippet(nil))
	require.Equal(t, "empty body", diagnosticBodySnippet([]byte("  \n\t ")))

	raw := "{a:1}\nb\"\xc2\xa0c\"\r\n"
	require.Equal(t, `{a:1} b" c"`, diagnosticBodySnippet([]byte(raw)))

	long := strings.Repeat("字", 500)
	snippet := diagnosticBodySnippet([]byte(long))
	require.Len(t, []rune(snippet), 301)
	require.True(t, strings.HasSuffix(snippet, "…"))
}

func TestFetchUpstreamSupportedModelsSurfaces401403BodyOnly(t *testing.T) {
	newSyncAccount := func() *Account {
		return &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://relay.example.com/v1",
			},
		}
	}

	t.Run("403 includes upstream reason", func(t *testing.T) {
		upstream := &queuedHTTPUpstream{responses: []*http.Response{
			newJSONResponse(http.StatusForbidden,
				`{"error":{"message":"该令牌不允许在此 IP 使用","type":"new_api_error"}}`),
		}}
		svc := &AccountTestService{httpUpstream: upstream}
		_, err := svc.FetchUpstreamSupportedModels(context.Background(), newSyncAccount())
		var syncErr *UpstreamModelSyncError
		require.ErrorAs(t, err, &syncErr)
		require.Contains(t, syncErr.SafeMessage(), "HTTP 403")
		require.Contains(t, syncErr.SafeMessage(), "该令牌不允许在此 IP 使用")
	})

	t.Run("502 stays status-only", func(t *testing.T) {
		upstream := &queuedHTTPUpstream{responses: []*http.Response{
			newJSONResponse(http.StatusBadGateway, `{"error":"SECRET_TOKEN should not be exposed"}`),
		}}
		svc := &AccountTestService{httpUpstream: upstream}
		_, err := svc.FetchUpstreamSupportedModels(context.Background(), newSyncAccount())
		var syncErr *UpstreamModelSyncError
		require.ErrorAs(t, err, &syncErr)
		require.Contains(t, syncErr.SafeMessage(), "HTTP 502")
		require.NotContains(t, syncErr.SafeMessage(), "SECRET_TOKEN")
	})
}
