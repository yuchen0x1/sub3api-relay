package proxyutil

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBuildVLESSRequestHeader(t *testing.T) {
	id := uuid.MustParse("d9428888-122b-11ed-861d-0242ac120002")
	header, err := buildVLESSRequestHeader(id, "api.example.com", 443)
	require.NoError(t, err)
	require.Equal(t, byte(0), header[0])
	require.Equal(t, id[:], header[1:17])
	require.Equal(t, []byte{0, vlessCommandTCP, 1, 187, vlessAddressDomain, 15}, header[17:23])
	require.Equal(t, []byte("api.example.com"), header[23:])
}

func TestVLESSDialContextWebSocketTLS(t *testing.T) {
	id := uuid.MustParse("d9428888-122b-11ed-861d-0242ac120002")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := coderws.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close(coderws.StatusNormalClosure, "")
		stream := coderws.NetConn(context.Background(), ws, coderws.MessageBinary)
		defer stream.Close()
		header := make([]byte, 0, 64)
		buf := make([]byte, 64)
		n, err := stream.Read(buf)
		if err != nil {
			return
		}
		header = append(header, buf[:n]...)
		if len(header) < 23 || header[0] != vlessVersion || header[17] != 0 || header[18] != vlessCommandTCP {
			return
		}
		port := binary.BigEndian.Uint16(header[19:21])
		if port != 80 {
			return
		}
		// VLESS response header (version + addon length), followed by payload.
		_, _ = stream.Write([]byte{vlessVersion, 0})
		_, _ = stream.Write([]byte("ok"))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	proxyURL := &url.URL{
		Scheme: "vless",
		Host:   serverURL.Host,
		User:   url.User(id.String()),
		RawQuery: strings.ReplaceAll(
			"security=tls&type=ws&sni=localhost&host=localhost&path=/vless&insecure=true",
			" ", "",
		),
	}
	dial, err := newVLESSDialContext(proxyURL)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := dial(ctx, "tcp", "example.com:80")
	require.NoError(t, err)
	defer conn.Close()
	payload, err := io.ReadAll(io.LimitReader(conn, 2))
	require.NoError(t, err)
	require.Equal(t, "ok", string(payload))
}
