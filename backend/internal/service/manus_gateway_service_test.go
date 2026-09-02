package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/tlsfingerprint"
)

type manusHTTPUpstreamRecorder struct {
	req         *http.Request
	proxyURL    string
	accountID   int64
	concurrency int
}

func (r *manusHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, concurrency int) (*http.Response, error) {
	r.req, r.proxyURL, r.accountID, r.concurrency = req, proxyURL, accountID, concurrency
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
}

func (r *manusHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return r.Do(req, proxyURL, accountID, concurrency)
}

func TestManusGatewayForwardAPIKeyStripsClientCredentialsAndUsesProxy(t *testing.T) {
	recorder := &manusHTTPUpstreamRecorder{}
	svc := NewManusGatewayService(recorder, &config.Config{Security: config.SecurityConfig{}}, nil)
	proxyID := int64(9)
	account := &Account{
		ID: 42, Platform: PlatformManus, Type: AccountTypeAPIKey, Concurrency: 3,
		Credentials: map[string]any{"base_url": "http://manus.test", "api_key": "server-key"},
		ProxyID:     &proxyID, Proxy: &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 8080},
	}
	headers := http.Header{
		"Authorization":   []string{"Bearer client-token"},
		"X-Manus-Api-Key": []string{"client-key"},
		"Cookie":          []string{"session=client"},
		"Content-Type":    []string{"application/json"},
	}

	resp, err := svc.Forward(t.Context(), account, http.MethodPost, "task.create", "locale=zh-CN", headers, strings.NewReader(`{"message":{"content":"hi"}}`))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "http://manus.test/v2/task.create?locale=zh-CN", recorder.req.URL.String())
	require.Equal(t, "server-key", recorder.req.Header.Get("x-manus-api-key"))
	require.Empty(t, recorder.req.Header.Get("Authorization"))
	require.Empty(t, recorder.req.Header.Get("Cookie"))
	require.Equal(t, "http://127.0.0.1:8080", recorder.proxyURL)
	require.Equal(t, int64(42), recorder.accountID)
	require.Equal(t, 3, recorder.concurrency)
}

func TestManusGatewayForwardOAuthUsesBearerOnly(t *testing.T) {
	recorder := &manusHTTPUpstreamRecorder{}
	svc := NewManusGatewayService(recorder, &config.Config{Security: config.SecurityConfig{}}, nil)
	account := &Account{ID: 7, Platform: PlatformManus, Type: AccountTypeOAuth, Credentials: map[string]any{"base_url": "http://manus.test", "access_token": "oauth-token"}}

	_, err := svc.Forward(t.Context(), account, http.MethodGet, "task.list", "limit=1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, "Bearer oauth-token", recorder.req.Header.Get("Authorization"))
	require.Empty(t, recorder.req.Header.Get("x-manus-api-key"))
}

func TestManusGatewayForwardRejectsInvalidEndpointAndMissingCredential(t *testing.T) {
	recorder := &manusHTTPUpstreamRecorder{}
	svc := NewManusGatewayService(recorder, &config.Config{Security: config.SecurityConfig{}}, nil)
	account := &Account{Platform: PlatformManus, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "http://manus.test"}}

	_, err := svc.Forward(t.Context(), account, http.MethodGet, "../task.list", "", nil, nil)
	require.ErrorContains(t, err, "unsupported Manus endpoint")
	_, err = svc.Forward(t.Context(), account, http.MethodGet, "task.list", "", nil, nil)
	require.ErrorContains(t, err, "API key is missing")
	require.Nil(t, recorder.req)
}

func TestAccountManusCredentialHelpers(t *testing.T) {
	account := &Account{Platform: PlatformManus, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": " key ", "base_url": "https://relay.example/"}}
	require.True(t, account.IsManus())
	require.Equal(t, "key", account.GetManusAPIKey())
	require.Equal(t, "https://relay.example", account.GetManusBaseURL())
	require.Empty(t, account.GetManusAccessToken())

	account.Type = AccountTypeOAuth
	account.Credentials["access_token"] = " token "
	require.Equal(t, "token", account.GetManusAccessToken())
	require.Empty(t, account.GetManusAPIKey())
}
