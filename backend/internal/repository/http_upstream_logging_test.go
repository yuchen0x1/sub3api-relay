package repository

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/tlsfingerprint"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestTLSFingerprintLogsDoNotExposeProxyCredentials(t *testing.T) {
	for _, isolation := range []string{config.ConnectionPoolIsolationProxy, config.ConnectionPoolIsolationAccountProxy} {
		t.Run(isolation, func(t *testing.T) {
			var output bytes.Buffer
			previous := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
			t.Cleanup(func() { slog.SetDefault(previous) })
			cfg := &config.Config{Gateway: config.GatewayConfig{ConnectionPoolIsolation: isolation}}
			s, ok := NewHTTPUpstream(cfg).(*httpUpstreamService)
			require.True(t, ok)
			profile := &tlsfingerprint.Profile{Name: "test"}
			proxyURL := "http://proxy-user:proxy-password@proxy.example:8080"
			entry, err := s.getClientEntryWithTLS(proxyURL, 42, 1, profile, service.HTTPUpstreamProfileDefault, false, false)
			require.NoError(t, err)
			reused, err := s.getClientEntryWithTLS(proxyURL, 42, 1, profile, service.HTTPUpstreamProfileDefault, false, false)
			require.NoError(t, err)
			require.Same(t, entry, reused)
			entry.poolKey = "stale"
			_, err = s.getClientEntryWithTLS(proxyURL, 42, 1, profile, service.HTTPUpstreamProfileDefault, false, false)
			require.NoError(t, err)
			got := output.String()
			for _, secret := range []string{"proxy-user", "proxy-password"} {
				require.NotContains(t, got, secret)
			}
			for _, diagnostic := range []string{"tls_fingerprint_creating_new_client", "tls_fingerprint_reusing_client", "tls_fingerprint_evicting_stale_client", "account_id=42", "proxy.example:8080"} {
				require.Contains(t, got, diagnostic)
			}
			// Connection isolation must still use the original credential-bearing key.
			require.Contains(t, entry.proxyKey, "proxy-password")
		})
	}
}

func TestTLSFingerprintAcquireErrorLogDoesNotExposeMalformedProxy(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	req, err := http.NewRequest(http.MethodGet, "https://upstream.example", nil)
	require.NoError(t, err)
	_, err = NewHTTPUpstream(nil).DoWithTLS(req, "http://proxy-user:proxy-password@bad%host", 42, 1, &tlsfingerprint.Profile{Name: "test"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid proxy URL")
	require.NotContains(t, output.String(), "proxy-user")
	require.NotContains(t, output.String(), "proxy-password")
	require.Contains(t, output.String(), "tls_fingerprint_acquire_client_failed")
	require.Contains(t, output.String(), "type=")
}

func TestRedactProxyForLogExcludesAllCredentialContainers(t *testing.T) {
	for _, raw := range []string{
		"http://user:password@proxy.example:8080/path-secret?key=query-secret#fragment-secret",
		"vless://uuid-secret@proxy.example:443?flow=query-secret",
		"trojan://password-secret@proxy.example:443?sni=example.com",
		"vmess://encoded-node-containing-credentials",
		"http://user:password@bad%host",
	} {
		got := redactProxyForLog(raw)
		for _, secret := range []string{"user", "password", "uuid-secret", "path-secret", "query-secret", "fragment-secret", "encoded-node-containing-credentials"} {
			if strings.Contains(got, secret) {
				t.Fatalf("proxy log %q leaked %q", got, secret)
			}
		}
	}
	require.Equal(t, "http://proxy.example:8080", redactProxyForLog("http://user:password@proxy.example:8080/path?key=secret"))
}
