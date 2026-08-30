package admin

import (
	"encoding/base64"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimSubscriptionURLPunctuation(t *testing.T) {
	require.Equal(t, "https://example.com/sub", trimSubscriptionURLPunctuation(" https://example.com/sub， "))
}

func TestParseProxySubscriptionBodyBase64Hysteria2(t *testing.T) {
	content := "hysteria2://secret@example.com:8443/?insecure=1#node-one\n"
	nodes, err := parseProxySubscriptionBody([]byte(base64.StdEncoding.EncodeToString([]byte(content))))
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "node-one", nodes[0].Name)
	require.Equal(t, "example.com", nodes[0].Host)
	require.Equal(t, 8443, nodes[0].Port)
	require.Equal(t, "secret", nodes[0].Password)
	require.True(t, nodes[0].TLSInsecure)
}

func TestParseProxySubscriptionRejectsUnsupportedHysteriaOptions(t *testing.T) {
	_, err := parseProxySubscriptionBody([]byte("hysteria2://secret@example.com:443/?obfs=salamander"))
	require.Error(t, err)
}

func TestParseProxySubscriptionBodyBase64VLESSWebSocket(t *testing.T) {
	content := "vless://d9428888-122b-11ed-861d-0242ac120002@edge.example.com:443?encryption=none&security=tls&type=ws&sni=cdn.example.com&host=cdn.example.com&path=%2Fnewlogin%2Flogin.do&fp=safari#Japan\n"
	nodes, err := parseProxySubscriptionBody([]byte(base64.StdEncoding.EncodeToString([]byte(content))))
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "vless", nodes[0].Protocol)
	require.Equal(t, "d9428888-122b-11ed-861d-0242ac120002", nodes[0].Username)
	require.Empty(t, nodes[0].Password)
	require.Equal(t, "/newlogin/login.do", nodes[0].Options["path"])
	require.Equal(t, "cdn.example.com", nodes[0].Options["sni"])
	require.Equal(t, "Japan", nodes[0].Name)
}

func TestParseVLESSSubscriptionRejectsUnsupportedTransport(t *testing.T) {
	_, err := parseProxySubscriptionBody([]byte("vless://d9428888-122b-11ed-861d-0242ac120002@example.com:443?security=reality&type=tcp"))
	require.Error(t, err)
}

func TestBlockedSubscriptionIP(t *testing.T) {
	blocked := []string{
		"0.0.0.0",
		"127.0.0.1",
		"10.0.0.1",
		"100.64.0.1",
		"169.254.169.254",
		"224.0.0.1",
		"192.0.2.1",
		"198.18.0.1",
		"198.51.100.1",
		"203.0.113.1",
		"240.0.0.1",
		"::1",
		"fc00::1",
		"fe80::1",
		"ff02::1",
		"64:ff9b::1",
		"64:ff9b:1::1",
		"100::1",
		"2001::1",
		"2001:db8::1",
		"2001:10::1",
		"2002::1",
		"3fff::1",
		"5f00::1",
	}
	for _, raw := range blocked {
		require.Truef(t, isBlockedSubscriptionIP(net.ParseIP(raw)), "%s must be blocked", raw)
	}
	require.False(t, isBlockedSubscriptionIP(net.ParseIP("8.8.8.8")))
	require.False(t, isBlockedSubscriptionIP(net.ParseIP("2606:4700:4700::1111")))
}

func TestSameSubscriptionOrigin(t *testing.T) {
	base, err := url.Parse("https://subscription.example/path/token")
	require.NoError(t, err)

	for _, raw := range []string{
		"https://subscription.example/next",
		"https://SUBSCRIPTION.EXAMPLE/next",
	} {
		target, parseErr := url.Parse(raw)
		require.NoError(t, parseErr)
		require.True(t, sameSubscriptionOrigin(base, target), raw)
	}
	for _, raw := range []string{
		"https://other.example/next",
		"https://subscription.example:8443/next",
		"http://subscription.example/next",
	} {
		target, parseErr := url.Parse(raw)
		require.NoError(t, parseErr)
		require.False(t, sameSubscriptionOrigin(base, target), raw)
	}
}
