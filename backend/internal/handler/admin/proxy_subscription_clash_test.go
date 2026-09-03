package admin

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const clashYAMLFixture = `mode: rule
proxies:
  - { name: sg-ss, type: ss, server: 1.2.3.4, port: 8388, cipher: aes-256-gcm, password: pass1 }
  - { name: sg-trojan, type: trojan, server: 5.6.7.8, port: 443, password: pw2, sni: example.com, skip-cert-verify: true }
  - { name: jp-vmess-ws, type: vmess, server: 9.9.9.9, port: 443, uuid: 1c1e4a5e-b47b-4c47-8a6b-7d0b8e5f3a01, alterId: 0, cipher: auto, tls: true, servername: v.example.com, network: ws, ws-opts: { path: /ws, headers: { Host: cdn.example.com } } }
  - { name: us-vless, type: vless, server: 7.7.7.7, port: 443, uuid: 2a2e4a5e-b47b-4c47-8a6b-7d0b8e5f3a02, tls: true, network: ws, servername: w.example.com, ws-opts: { path: /vless } }
  - { name: hk-hy2, type: hysteria2, server: 4.4.4.4, port: 443, password: hy2pass }
  - { name: sg-vmess-old, type: vmess, server: 6.6.6.6, port: 80, uuid: 3a3e4a5e-b47b-4c47-8a6b-7d0b8e5f3a03, alterId: 7, cipher: auto, network: tcp }
`

func TestParseClashProxyYAML(t *testing.T) {
	nodes, skipped, ok := parseClashProxyYAML(clashYAMLFixture)
	require.True(t, ok)
	require.Len(t, nodes, 4)
	require.Equal(t, 1, skipped["unsupported node type hysteria2"])
	require.Equal(t, 1, skipped["vmess AEAD requires alterId 0"])

	byName := map[string]parsedProxySubscription{}
	for _, node := range nodes {
		byName[node.Name] = node
	}

	ss := byName["sg-ss"]
	require.Equal(t, "ss", ss.Protocol)
	require.Equal(t, "1.2.3.4", ss.Host)
	require.Equal(t, 8388, ss.Port)
	require.Equal(t, "aes-256-gcm", ss.Username)
	require.Equal(t, "pass1", ss.Password)

	trojan := byName["sg-trojan"]
	require.Equal(t, "trojan", trojan.Protocol)
	require.Equal(t, "pw2", trojan.Username)
	require.Equal(t, "example.com", trojan.Options["sni"])
	require.Equal(t, "true", trojan.Options["allowInsecure"])

	vmess := byName["jp-vmess-ws"]
	require.Equal(t, "vmess", vmess.Protocol)
	require.Equal(t, "9.9.9.9", vmess.Host)
	require.Equal(t, "ws", vmess.Options["net"])
	require.Equal(t, "tls", vmess.Options["tls"])
	require.Equal(t, "/ws", vmess.Options["path"])
	require.Equal(t, "cdn.example.com", vmess.Options["host"])
	require.Equal(t, "v.example.com", vmess.Options["sni"])

	vless := byName["us-vless"]
	require.Equal(t, "vless", vless.Protocol)
	require.Equal(t, "7.7.7.7", vless.Host)
	require.Equal(t, "tls", vless.Options["security"])
	require.Equal(t, "ws", vless.Options["type"])
	require.Equal(t, "/vless", vless.Options["path"])
}

func TestParseClashProxyYAMLRejectsNonClash(t *testing.T) {
	_, _, ok := parseClashProxyYAML("vless://00000000-0000-0000-0000-000000000000@example.com:443?type=ws#x")
	require.False(t, ok)
	_, _, ok = parseClashProxyYAML("just some text")
	require.False(t, ok)
}

func TestParseProxySubscriptionBodyClashYAML(t *testing.T) {
	nodes, skipped, err := parseProxySubscriptionBody([]byte(clashYAMLFixture))
	require.NoError(t, err)
	require.Len(t, nodes, 4)
	require.Equal(t, 1, skipped["unsupported node type hysteria2"])
}

func TestParseProxySubscriptionBodyClashBase64(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(clashYAMLFixture))
	nodes, _, err := parseProxySubscriptionBody([]byte(encoded))
	require.NoError(t, err)
	require.Len(t, nodes, 4)
}

func TestFetchSubscriptionWithClientUARetries(t *testing.T) {
	seen := make([]string, 0, 2)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("User-Agent"))
		if r.Header.Get("User-Agent") == proxySubscriptionPrimaryUA {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
			return
		}
		_, _ = w.Write([]byte("proxies: []"))
	}))
	defer server.Close()

	httpReq, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := fetchSubscriptionWithClientUA(server.Client(), httpReq)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []string{proxySubscriptionPrimaryUA, proxySubscriptionFallbackUA}, seen)
}

func TestFetchSubscriptionWithClientUAGivesUpAfterFallback(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	httpReq, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	_, err = fetchSubscriptionWithClientUA(server.Client(), httpReq)
	require.EqualError(t, err, "subscription returned HTTP 403")
	_ = time.Second
}
