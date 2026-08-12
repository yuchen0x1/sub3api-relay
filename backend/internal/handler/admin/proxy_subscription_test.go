package admin

import (
	"encoding/base64"
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

func TestBlockedSubscriptionIP(t *testing.T) {
	require.True(t, isBlockedSubscriptionIP([]byte{127, 0, 0, 1}))
}
