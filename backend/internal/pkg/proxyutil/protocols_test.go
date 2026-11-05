package proxyutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"hash/fnv"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// --- node parsing ---

func TestParseSSNodeBase64Userinfo(t *testing.T) {
	node, err := ParseSSNode("ss://YWVzLTI1Ni1nY206MTIzNDU2@0.0.0.0:1#node")
	require.NoError(t, err)
	require.Equal(t, "aes-256-gcm", node.Method)
	require.Equal(t, "123456", node.Password)
	require.Equal(t, "0.0.0.0", node.Host)
	require.Equal(t, 1, node.Port)
}

func TestParseSSNodePlainUserinfo(t *testing.T) {
	node, err := ParseSSNode("ss://chacha20-ietf-poly1305:secret%2Fkey@relay.example.com:8388#x")
	require.NoError(t, err)
	require.Equal(t, "chacha20-ietf-poly1305", node.Method)
	require.Equal(t, "secret/key", node.Password)
}

func TestParseSSNodeRejectsUnsupportedCipherAndPlugin(t *testing.T) {
	_, err := ParseSSNode("ss://cmM0LW1kNTpwYXNz@relay.example.com:8388")
	require.ErrorContains(t, err, "unsupported shadowsocks method")
	_, err = ParseSSNode("ss://YWVzLTI1Ni1nY206cHc@relay.example.com:8388?plugin=obfs-local")
	require.ErrorContains(t, err, "plugin")
	_, err = ParseSSNode("ss://YWVzLTI1Ni1nY206cHc@relay.example.com:99999")
	require.ErrorContains(t, err, "port")
}

func TestParseTrojanNode(t *testing.T) {
	node, err := ParseTrojanNode("trojan://pass%40word@gw.example.com:443?sni=real.example.com&allowInsecure=1#edge")
	require.NoError(t, err)
	require.Equal(t, "pass@word", node.Password)
	require.Equal(t, "gw.example.com", node.Host)
	require.Equal(t, 443, node.Port)
	require.Equal(t, "real.example.com", node.SNI)
	require.True(t, node.Insecure)
}

func TestParseVMESSNodeJSON(t *testing.T) {
	fields := map[string]string{
		"v": "2", "add": "vm.example.com", "port": "8443", "id": "d9428888-122b-11ed-861d-0242ac120002",
		"aid": "0", "scy": "auto", "net": "ws", "host": "cdn.example.com", "path": "/ws",
		"tls": "tls", "sni": "sni.example.com",
	}
	encoded, err := json.Marshal(fields)
	require.NoError(t, err)
	node, err := ParseVMESSNode("vmess://" + base64EncodeForTest(encoded))
	require.NoError(t, err)
	require.Equal(t, "d9428888-122b-11ed-861d-0242ac120002", node.UUID)
	require.Equal(t, "aes-128-gcm", node.Cipher)
	require.Equal(t, "ws", node.Network)
	require.True(t, node.TLS)
	require.Equal(t, "cdn.example.com", node.Host)
	require.Equal(t, "/ws", node.Path)
	require.Equal(t, "sni.example.com", node.SNI)
	require.Equal(t, 8443, node.ServerPort)
}

func TestParseVMESSNodeRejectsInvalid(t *testing.T) {
	mk := func(fields map[string]string) string {
		encoded, err := json.Marshal(fields)
		require.NoError(t, err)
		return "vmess://" + base64EncodeForTest(encoded)
	}
	_, err := ParseVMESSNode(mk(map[string]string{"add": "h", "port": "443", "id": "not-a-uuid"}))
	require.ErrorContains(t, err, "UUID")
	_, err = ParseVMESSNode(mk(map[string]string{"add": "h", "port": "443", "id": "d9428888-122b-11ed-861d-0242ac120002", "aid": "64"}))
	require.ErrorContains(t, err, "alterId")
	_, err = ParseVMESSNode(mk(map[string]string{"add": "h", "port": "443", "id": "d9428888-122b-11ed-861d-0242ac120002", "net": "grpc"}))
	require.ErrorContains(t, err, "transport")
	_, err = ParseVMESSNode("vmess://not-base64-json!!")
	require.Error(t, err)
}

func TestSerializeVMESSURLRoundTrip(t *testing.T) {
	options := map[string]string{
		"net": "ws", "host": "cdn.example.com", "path": "/wspath", "tls": "tls",
		"sni": "sni.example.com", "scy": "aes-128-gcm",
	}
	raw := SerializeVMESSURL("d9428888-122b-11ed-861d-0242ac120002", "0", "vm.example.com", "8443", options)
	node, err := ParseVMESSNode(raw)
	require.NoError(t, err)
	require.Equal(t, "d9428888-122b-11ed-861d-0242ac120002", node.UUID)
	require.Equal(t, "vm.example.com", node.ServerHost)
	require.Equal(t, 8443, node.ServerPort)
	require.Equal(t, "ws", node.Network)
	require.True(t, node.TLS)
	require.Equal(t, "/wspath", node.Path)
	require.Equal(t, "cdn.example.com", node.Host)
	require.Equal(t, "sni.example.com", node.SNI)
}

// --- dialer dispatch accepts the new schemes ---

func TestConfigureTransportProxyAcceptsNewSchemes(t *testing.T) {
	cases := []string{
		"ss://YWVzLTI1Ni1nY206cHc@relay.example.com:8388",
		"trojan://password@gw.example.com:443",
		"vmess://" + base64EncodeForTest([]byte(`{"v":"2","add":"vm.example.com","port":"443","id":"d9428888-122b-11ed-861d-0242ac120002","aid":"0","net":"tcp"}`)),
	}
	for _, raw := range cases {
		proxyURL, err := url.Parse(raw)
		require.NoError(t, err)
		transport := &http.Transport{}
		require.NoErrorf(t, ConfigureTransportProxy(transport, proxyURL), "%s", raw)
		require.Nil(t, transport.Proxy)
		require.NotNilf(t, transport.DialContext, "%s", raw)
	}
}

// --- shared test helpers (values are throwaway vectors computed at runtime,
// not credentials) ---

func base64EncodeForTest(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func derivedTestSecret(seed string) string {
	sum := sha256.Sum256([]byte("sub3api-protocol-test:" + seed))
	return "v1-" + hexEncodeForTest(sum[:8])
}

func hexEncodeForTest(data []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(data)*2)
	for _, b := range data {
		out = append(out, digits[b>>4], digits[b&0x0F])
	}
	return string(out)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}

func mustParseUUIDForTest(raw string) uuid.UUID {
	id, err := uuid.Parse(raw)
	if err != nil {
		panic("test uuid invalid: " + err.Error())
	}
	return id
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return cert
}

// --- Shadowsocks end-to-end against a local AEAD server ---

func startFakeSSListener(t *testing.T, method string, keyLen int, password string) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			t.Logf("fake ss: accepted connection from %s", conn.RemoteAddr())
			go handleFakeSSConn(t, conn, method, keyLen, password)
		}
	}()
	return listener
}

func handleFakeSSConn(t *testing.T, conn net.Conn, method string, keyLen int, password string) {
	defer func() { _ = conn.Close() }()
	newAEAD := ssCiphers[method].newAEAD
	master := ssKDF(password, keyLen)
	salt := make([]byte, keyLen)
	if _, err := io.ReadFull(conn, salt); err != nil {
		t.Logf("fake ss: read salt: %v", err)
		return
	}
	readAEAD, err := newAEAD(ssSubkey(master, salt, keyLen))
	if err != nil {
		t.Logf("fake ss: aead: %v", err)
		return
	}
	readKey := &ssStreamKey{aead: readAEAD}

	var lengthSealed [2 + 16]byte
	if _, err := io.ReadFull(conn, lengthSealed[:]); err != nil {
		t.Logf("fake ss: read length: %v", err)
		return
	}
	lengthPlain, err := readAEAD.Open(nil, readKey.nextNonce(), lengthSealed[:], nil)
	if err != nil {
		t.Logf("fake ss: open length: %v", err)
		return
	}
	target := make([]byte, int(binary.BigEndian.Uint16(lengthPlain))+16)
	if _, err := io.ReadFull(conn, target); err != nil {
		t.Logf("fake ss: read target: %v", err)
		return
	}
	if _, err := readAEAD.Open(nil, readKey.nextNonce(), target, nil); err != nil {
		t.Logf("fake ss: open target: %v", err)
		return
	}
	t.Logf("fake ss: target decrypted ok (%d bytes), sending response", len(target))

	writeSalt := make([]byte, keyLen)
	_, _ = rand.Read(writeSalt)
	if _, err := conn.Write(writeSalt); err != nil {
		t.Logf("fake ss: write salt: %v", err)
		return
	}
	writeAEAD, err := newAEAD(ssSubkey(master, writeSalt, keyLen))
	if err != nil {
		return
	}
	writeKey := &ssStreamKey{aead: writeAEAD}
	httpResponse := []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok")
	if err := writeKey.writeChunk(conn, httpResponse); err != nil {
		t.Logf("fake ss: write chunk: %v", err)
		return
	}
	// Keep the connection open until the client is done reading.
	time.Sleep(3 * time.Second)
}

func TestShadowsocksDialerEndToEnd(t *testing.T) {
	const method = "aes-256-gcm"
	password := derivedTestSecret("ss")
	listener := startFakeSSListener(t, method, 32, password)
	defer func() { _ = listener.Close() }()

	proxyURL, err := url.Parse("ss://" + method + ":" + password + "@" + listener.Addr().String())
	require.NoError(t, err)
	transport := &http.Transport{}
	require.NoError(t, ConfigureTransportProxy(transport, proxyURL))

	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get("http://target.example.com/anything")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "ok", string(body))
}

// --- Trojan end-to-end against a local TLS server ---

func TestTrojanDialerEndToEnd(t *testing.T) {
	password := derivedTestSecret("trojan")
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert := selfSignedCert(t)
			return &cert, nil
		},
	})
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(raw net.Conn) {
				defer func() { _ = raw.Close() }()
				buf := make([]byte, 4096)
				// auth hash + CRLF + address + CRLF (single read suffices locally)
				if _, err := raw.Read(buf); err != nil {
					return
				}
				if _, err := raw.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\ntl")); err != nil {
					return
				}
			}(conn)
		}
	}()

	proxyURL, err := url.Parse("trojan://" + password + "@" + listener.Addr().String() + "?allowInsecure=1")
	require.NoError(t, err)
	transport := &http.Transport{}
	require.NoError(t, ConfigureTransportProxy(transport, proxyURL))

	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get("http://target.example.com/anything")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, 200, resp.StatusCode)
}

// --- VMess end-to-end against a local AEAD server ---

func startFakeVMESSListener(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleFakeVMESSConn(conn)
		}
	}()
	return listener
}

func handleFakeVMESSConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	const (
		saltHeaderLenKey = "VMess Header AEAD Key_Length"
		saltHeaderLenIV  = "VMess Header AEAD Nonce_Length"
		saltHeaderKey    = "VMess Header AEAD Key"
		saltHeaderIV     = "VMess Header AEAD Nonce"
		saltRespLenKey   = "AEAD Resp Header Len Key"
		saltRespLenIV    = "AEAD Resp Header Len IV"
		saltRespKey      = "AEAD Resp Header Key"
		saltRespIV       = "AEAD Resp Header IV"
		testUUID         = "d9428888-122b-11ed-861d-0242ac120002"
	)
	cmdKey := vmessCmdKey(mustParseUUIDForTest(testUUID))

	var authID [16]byte
	if _, err := io.ReadFull(conn, authID[:]); err != nil {
		return
	}
	var sealedLength [18]byte
	if _, err := io.ReadFull(conn, sealedLength[:]); err != nil {
		return
	}
	var nonce [8]byte
	if _, err := io.ReadFull(conn, nonce[:]); err != nil {
		return
	}
	lengthGCM := vmessAESGCM(vmessKDF16(cmdKey, saltHeaderLenKey, string(authID[:]), string(nonce[:])))
	lengthPlain, err := lengthGCM.Open(nil, vmessKDF(cmdKey, saltHeaderLenIV, string(authID[:]), string(nonce[:]))[:12], sealedLength[:], authID[:])
	if err != nil {
		return
	}
	headerLength := int(binary.BigEndian.Uint16(lengthPlain))
	sealedHeader := make([]byte, headerLength+16)
	if _, err := io.ReadFull(conn, sealedHeader); err != nil {
		return
	}
	headerGCM := vmessAESGCM(vmessKDF16(cmdKey, saltHeaderKey, string(authID[:]), string(nonce[:])))
	header, err := headerGCM.Open(nil, vmessKDF(cmdKey, saltHeaderIV, string(authID[:]), string(nonce[:]))[:12], sealedHeader, authID[:])
	if err != nil {
		return
	}
	// header: [1 ver][16 IV][16 key][1 respAuth][1 opt][1 sec][1 rsv][1 cmd][addr][FNV4]
	if len(header) < 42 || header[0] != 1 {
		return
	}
	reqIV := header[1:17]
	reqKey := header[17:33]
	respAuth := header[33]
	opt := header[34]
	if opt&vmessOptChunkStream == 0 || opt&vmessOptChunkMasking == 0 {
		return
	}
	fnv1a := fnv.New32a()
	_, _ = fnv1a.Write(header[:len(header)-4])
	if binary.BigEndian.Uint32(header[len(header)-4:]) != fnv1a.Sum32() {
		return
	}

	reqKeyHash := sha256.Sum256(reqKey)
	reqIVHash := sha256.Sum256(reqIV)
	respKey := reqKeyHash[:16]
	respIV := reqIVHash[:16]
	respHeaderPayload := []byte{respAuth, vmessOptChunkStream | vmessOptChunkMasking}
	// Only the two response-header layers use KDF-derived keys; the body
	// cipher is the raw response key per the spec.
	respLenGCM := vmessAESGCM(vmessKDF16(respKey, saltRespLenKey))
	respLenIV := vmessKDF(respIV, saltRespLenIV)[:12]
	respHeaderGCM := vmessAESGCM(vmessKDF16(respKey, saltRespKey))
	respHeaderIV := vmessKDF(respIV, saltRespIV)[:12]
	respBodyGCM := vmessAESGCM(respKey)

	out := respLenGCM.Seal(nil, respLenIV, []byte{0, byte(len(respHeaderPayload))}, nil)
	out = append(out, respHeaderGCM.Seal(nil, respHeaderIV, respHeaderPayload, nil)...)
	if _, err := conn.Write(out); err != nil {
		return
	}

	// body: masked chunks under respKey/respIV.
	respMask := vmessShakeMask(respIV)
	writeChunk := func(counter uint16, payload []byte) bool {
		mask := nextMaskForTest(respMask)
		var sizeBytes [2]byte
		binary.BigEndian.PutUint16(sizeBytes[:], uint16(len(payload))^mask)
		sealed := respBodyGCM.Seal(nil, vmessChunkNonce(respIV, counter), payload, nil)
		if _, err := conn.Write(sizeBytes[:]); err != nil {
			return false
		}
		_, err := conn.Write(sealed)
		return err == nil
	}
	if !writeChunk(0, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nvm")) {
		return
	}
}

func nextMaskForTest(shake interface{ Read([]byte) (int, error) }) uint16 {
	var mask [2]byte
	_, _ = shake.Read(mask[:])
	return binary.BigEndian.Uint16(mask[:])
}

func TestVMESSDialerEndToEnd(t *testing.T) {
	listener := startFakeVMESSListener(t)
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	fields := map[string]string{
		"v": "2", "add": "127.0.0.1", "port": strconv.Itoa(addr.Port),
		"id": "d9428888-122b-11ed-861d-0242ac120002", "aid": "0", "scy": "aes-128-gcm", "net": "tcp",
	}
	encoded, err := json.Marshal(fields)
	require.NoError(t, err)

	transport := &http.Transport{}
	require.NoError(t, ConfigureTransportProxy(transport, mustParseURL(t, "vmess://"+base64EncodeForTest(encoded))))

	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get("http://target.example.com/anything")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "vm", string(body))
}
