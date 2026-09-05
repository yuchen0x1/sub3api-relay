package proxyutil

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/sha3"
)

// VMess client implementing the AEAD request header (v2ray V2 spec) with
// AES-128-GCM / ChaCha20-Poly1305 payload, over TCP (optionally TLS-wrapped)
// or WebSocket (ws/wss). AlterID must be 0: legacy non-AEAD handshakes are
// rejected by modern servers and are not implemented.

const (
	vmessKDFSaltAuthIDKey         = "AES Auth ID Encryption"
	vmessKDFSaltHeaderLengthKey   = "VMess Header AEAD Key_Length"
	vmessKDFSaltHeaderLengthIV    = "VMess Header AEAD Nonce_Length"
	vmessKDFSaltHeaderKey         = "VMess Header AEAD Key"
	vmessKDFSaltHeaderIV          = "VMess Header AEAD Nonce"
	vmessKDFSaltRespLengthKey     = "AEAD Resp Header Len Key"
	vmessKDFSaltRespLengthIV      = "AEAD Resp Header Len IV"
	vmessKDFSaltRespPayloadKey    = "AEAD Resp Header Key"
	vmessKDFSaltRespPayloadIV     = "AEAD Resp Header IV"
	vmessChunkMaxPayload          = 0x3FFF
	vmessOptChunkStream           = 0x01
	vmessOptChunkMasking          = 0x04
	vmessSecurityAES128GCM        = 0x03
	vmessSecurityChaCha20Poly1305 = 0x04
	vmessCommandTCP               = 0x01
	vmessAddressIPv4              = 1
	vmessAddressDomain            = 2
	vmessAddressIPv6              = 3
	vmessTCPHandshakeBound        = 10 * time.Second
)

type vmessConfig struct {
	UUID       uuid.UUID
	Cipher     string // "aes-128-gcm" or "chacha20-ietf-poly1305"
	Network    string // "ws" or "tcp"
	TLS        bool
	SNI        string
	Host       string
	Path       string
	Insecure   bool
	ServerHost string
	ServerPort string
}

// ParseVMESSConfig decodes a vmess:// URI in the standard base64(JSON) form.
func ParseVMESSConfig(raw string) (*vmessConfig, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "vmess") {
		return nil, errors.New("invalid vmess proxy URL")
	}
	payload := parsed.Host + parsed.Opaque + parsed.Path
	compact := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "", "=", "").Replace(payload)
	decoded, err := base64.RawStdEncoding.DecodeString(strings.NewReplacer("-", "+", "_", "/").Replace(compact))
	if err != nil {
		return nil, errors.New("vmess URI payload is not base64")
	}
	var fields map[string]any
	if err := json.Unmarshal(decoded, &fields); err != nil {
		return nil, errors.New("vmess URI payload is not a JSON node descriptor")
	}
	get := func(name string) string {
		switch v := fields[name].(type) {
		case string:
			return strings.TrimSpace(v)
		case float64:
			text := strconv.FormatFloat(v, 'f', -1, 64)
			return strings.TrimSuffix(text, ".0")
		default:
			return ""
		}
	}

	cfg := &vmessConfig{}
	id, err := uuid.Parse(get("id"))
	if err != nil {
		return nil, errors.New("vmess UUID is invalid")
	}
	cfg.UUID = id

	if raw := get("aid"); raw != "" {
		altID, err := strconv.Atoi(raw)
		if err != nil || altID != 0 {
			return nil, errors.New("vmess AEAD requires alterId 0")
		}
	}

	switch cipherName := strings.ToLower(get("scy")); cipherName {
	case "", "auto", "aes-128-gcm":
		cfg.Cipher = "aes-128-gcm"
	case "chacha20-ietf-poly1305", "chacha20-poly1305":
		cfg.Cipher = "chacha20-ietf-poly1305"
	default:
		return nil, fmt.Errorf("unsupported vmess cipher %q", cipherName)
	}

	cfg.Network = strings.ToLower(get("net"))
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Network != "ws" && cfg.Network != "tcp" {
		return nil, fmt.Errorf("unsupported vmess transport %q (supported: ws, tcp)", cfg.Network)
	}
	if strings.EqualFold(get("tls"), "reality") {
		return nil, errors.New("vmess reality is not supported")
	}
	cfg.TLS = strings.EqualFold(get("tls"), "tls")
	cfg.Host = get("host")
	cfg.Path = get("path")
	cfg.SNI = get("sni")
	switch strings.ToLower(get("skip-cert-verify")) {
	case "1", "true":
		cfg.Insecure = true
	}

	cfg.ServerHost = get("add")
	if cfg.ServerHost == "" {
		return nil, errors.New("vmess node is missing host")
	}
	port := get("port")
	if port == "" {
		port = "443"
	}
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
		return nil, errors.New("invalid vmess port")
	}
	cfg.ServerPort = port
	if cfg.Network == "ws" {
		if cfg.Path == "" {
			cfg.Path = "/"
		}
		if !strings.HasPrefix(cfg.Path, "/") {
			return nil, errors.New("invalid vmess websocket path")
		}
	}
	return cfg, nil
}

// newVMESSDialContext returns a DialContext tunneling TCP through VMess AEAD.
// rawConfig must be a vmess:// URI (ParseVMESSConfig validates it).
func newVMESSDialContext(rawConfig string) (func(context.Context, string, string) (net.Conn, error), error) {
	cfg, err := ParseVMESSConfig(rawConfig)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, network, destination string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("vmess only supports TCP dialing, got %s", network)
		}
		host, port, err := net.SplitHostPort(destination)
		if err != nil {
			return nil, fmt.Errorf("invalid vmess destination: %w", err)
		}
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return nil, errors.New("invalid vmess destination port")
		}

		underlying, err := dialVMESSUnderlying(ctx, cfg)
		if err != nil {
			return nil, err
		}

		session, err := newVMESSSession(cfg, underlying, host, portNumber)
		if err != nil {
			_ = underlying.Close()
			return nil, err
		}
		return session, nil
	}, nil
}

func dialVMESSUnderlying(ctx context.Context, cfg *vmessConfig) (net.Conn, error) {
	if cfg.Network == "ws" {
		return dialVMESSWebSocket(ctx, cfg)
	}
	var dialer net.Dialer
	rawConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.ServerHost, cfg.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("connect vmess server: %w", err)
	}
	if !cfg.TLS {
		return rawConn, nil
	}
	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         vmessSNI(cfg),
		InsecureSkipVerify: cfg.Insecure, //nolint:gosec // opt-in is carried by the subscription itself
		MinVersion:         tls.VersionTLS12,
	})
	handshakeCtx, cancel := context.WithTimeout(ctx, vmessTCPHandshakeBound)
	defer cancel()
	if err := tlsConn.HandshakeContext(handshakeCtx); err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("vmess TLS handshake failed: %w", err)
	}
	return tlsConn, nil
}

func vmessSNI(cfg *vmessConfig) string {
	if cfg.SNI != "" {
		return cfg.SNI
	}
	return cfg.ServerHost
}

func dialVMESSWebSocket(ctx context.Context, cfg *vmessConfig) (net.Conn, error) {
	wsScheme := "ws"
	if cfg.TLS {
		wsScheme = "wss"
	}
	wsPath := cfg.Path
	wsQuery := ""
	if queryIndex := strings.IndexByte(wsPath, '?'); queryIndex >= 0 {
		wsQuery = wsPath[queryIndex+1:]
		wsPath = wsPath[:queryIndex]
	}
	wsURL := url.URL{
		Scheme:   wsScheme,
		Host:     net.JoinHostPort(cfg.ServerHost, cfg.ServerPort),
		Path:     wsPath,
		RawQuery: wsQuery,
	}
	transport := &http.Transport{
		Proxy:               nil,
		DialContext:         (&net.Dialer{Timeout: vmessTCPHandshakeBound, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: vmessTCPHandshakeBound,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        1,
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     10 * time.Second,
	}
	if cfg.TLS {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         vmessSNI(cfg),
			InsecureSkipVerify: cfg.Insecure, //nolint:gosec // opt-in is carried by the subscription itself
		}
	}
	httpClient := &http.Client{Transport: transport}
	wsConn, resp, err := coderws.Dial(ctx, wsURL.String(), &coderws.DialOptions{
		HTTPClient: httpClient,
		Host:       cfg.Host,
		HTTPHeader: http.Header{"User-Agent": []string{"sub3api-vmess/1.0"}},
	})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("vmess WebSocket handshake failed with HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("vmess WebSocket handshake failed: %w", err)
	}
	// Same lifetime rule as VLESS: the request context bounds only the
	// handshake; the stream must survive after DialContext returns.
	return coderws.NetConn(context.Background(), wsConn, coderws.MessageBinary), nil
}

// vmessSession is a net.Conn speaking the VMess AEAD record layer.
type vmessSession struct {
	net.Conn
	cfg      *vmessConfig
	reqKey   [16]byte
	reqIV    [16]byte
	respAuth byte

	reqMask   sha3.ShakeHash
	reqChunks uint16

	respOpened bool
	respOpt    byte
	respMask   sha3.ShakeHash
	respChunks uint16
	respBuf    []byte
}

func newVMESSSession(cfg *vmessConfig, conn net.Conn, destHost string, destPort int) (*vmessSession, error) {
	session := &vmessSession{Conn: conn, cfg: cfg}
	if _, err := rand.Read(session.reqKey[:]); err != nil {
		return nil, fmt.Errorf("generate vmess request key: %w", err)
	}
	if _, err := rand.Read(session.reqIV[:]); err != nil {
		return nil, fmt.Errorf("generate vmess request IV: %w", err)
	}
	if _, err := rand.Read([]byte{session.respAuth}); err != nil {
		return nil, fmt.Errorf("generate vmess response marker: %w", err)
	}
	session.reqMask = vmessShakeMask(session.reqIV[:])

	instruction, err := buildVMESSInstruction(session, cfg, destHost, destPort)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(sealVMESSAEADHeader(vmessCmdKey(cfg.UUID), instruction)); err != nil {
		return nil, fmt.Errorf("write vmess request header: %w", err)
	}
	return session, nil
}

// buildVMESSInstruction assembles the decrypted request-header payload:
// version, session IV/key, response marker, options, security, command,
// target address and the trailing checksum.
func buildVMESSInstruction(session *vmessSession, cfg *vmessConfig, destHost string, destPort int) ([]byte, error) {
	buf := make([]byte, 0, 64+len(destHost))
	buf = append(buf, 0x01) // protocol version
	buf = append(buf, session.reqIV[:]...)
	buf = append(buf, session.reqKey[:]...)
	buf = append(buf, session.respAuth)
	buf = append(buf, vmessOptChunkStream|vmessOptChunkMasking)
	security := byte(vmessSecurityAES128GCM)
	if cfg.Cipher == "chacha20-ietf-poly1305" {
		security = vmessSecurityChaCha20Poly1305
	}
	buf = append(buf, security, 0x00, vmessCommandTCP)
	if err := appendVMessAddress(&buf, destHost, destPort); err != nil {
		return nil, err
	}
	fnv1a := fnv.New32a()
	_, _ = fnv1a.Write(buf)
	var checksum [4]byte
	binary.BigEndian.PutUint32(checksum[:], fnv1a.Sum32())
	return append(buf, checksum[:]...), nil
}

// appendVMessAddress appends ATYP + address + port (port last, big endian).
func appendVMessAddress(dst *[]byte, host string, port int) error {
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			*dst = append(*dst, vmessAddressIPv4)
			*dst = append(*dst, ip4...)
		} else {
			*dst = append(*dst, vmessAddressIPv6)
			*dst = append(*dst, ip.To16()...)
		}
	} else {
		if len(host) > 255 || strings.ContainsAny(host, "\x00\r\n") {
			return errors.New("invalid vmess destination host")
		}
		*dst = append(*dst, vmessAddressDomain, byte(len(host)))
		*dst = append(*dst, host...)
	}
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], uint16(port))
	*dst = append(*dst, portBytes[:]...)
	return nil
}

// sealVMESSAEADHeader produces EAuID(16) + ALength(18) + Nonce(8) + AHeader.
func sealVMESSAEADHeader(cmdKey []byte, instruction []byte) []byte {
	authID := createVMESSAuthID(cmdKey)
	nonce := make([]byte, 8)
	_, _ = randRead(nonce)

	lengthKey := vmessKDF16(cmdKey, vmessKDFSaltHeaderLengthKey, string(authID[:]), string(nonce))
	lengthIV := vmessKDF(cmdKey, vmessKDFSaltHeaderLengthIV, string(authID[:]), string(nonce))[:12]
	payloadKey := vmessKDF16(cmdKey, vmessKDFSaltHeaderKey, string(authID[:]), string(nonce))
	payloadIV := vmessKDF(cmdKey, vmessKDFSaltHeaderIV, string(authID[:]), string(nonce))[:12]

	var lengthPlain [2]byte
	binary.BigEndian.PutUint16(lengthPlain[:], uint16(len(instruction)))

	out := make([]byte, 0, 16+18+8+len(instruction)+16)
	out = append(out, authID[:]...)
	out = vmessAESGCM(lengthKey).Seal(out, lengthIV, lengthPlain[:], authID[:])
	out = append(out, nonce...)
	out = vmessAESGCM(payloadKey).Seal(out, payloadIV, instruction, authID[:])
	return out
}

func vmessShakeMask(iv []byte) sha3.ShakeHash {
	shake := sha3.NewShake128()
	_, _ = shake.Write(iv)
	return shake
}

func (s *vmessSession) nextMask(shake sha3.ShakeHash) uint16 {
	var mask [2]byte
	_, _ = shake.Read(mask[:])
	return binary.BigEndian.Uint16(mask[:])
}

// vmessChunkNonce: counter in the first 2 bytes, rest of the IV follows.
func vmessChunkNonce(iv []byte, count uint16) []byte {
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint16(nonce[:2], count)
	copy(nonce[2:], iv[2:12])
	return nonce
}

func (s *vmessSession) payloadAEAD() cipher.AEAD {
	if s.cfg.Cipher == "chacha20-ietf-poly1305" {
		aead, err := chacha20poly1305.New(vmessChaChaKey(s.reqKey[:]))
		if err != nil {
			panic("vmess chacha init failed: " + err.Error())
		}
		return aead
	}
	return vmessAESGCM(s.reqKey[:])
}

func (s *vmessSession) Write(p []byte) (int, error) {
	aead := s.payloadAEAD()
	total := 0
	for total < len(p) {
		end := total + vmessChunkMaxPayload
		if end > len(p) {
			end = len(p)
		}
		chunk := p[total:end]
		size := uint16(len(chunk)) ^ s.nextMask(s.reqMask)
		var header [2]byte
		binary.BigEndian.PutUint16(header[:], size)
		sealed := aead.Seal(nil, vmessChunkNonce(s.reqIV[:], s.reqChunks), chunk, nil)
		s.reqChunks++
		if _, err := s.Conn.Write(header[:]); err != nil {
			return total, fmt.Errorf("write vmess chunk length: %w", err)
		}
		if _, err := s.Conn.Write(sealed); err != nil {
			return total, fmt.Errorf("write vmess chunk payload: %w", err)
		}
		total = end
	}
	return total, nil
}

// openResponseHeader decrypts the AEAD response header on first read.
func (s *vmessSession) openResponseHeader() error {
	reqKey := sha256.Sum256(s.reqKey[:])
	reqIV := sha256.Sum256(s.reqIV[:])
	respKey := reqKey[:16]
	respIV := reqIV[:16]

	var encryptedLength [18]byte
	if _, err := io.ReadFull(s.Conn, encryptedLength[:]); err != nil {
		return fmt.Errorf("read vmess response header length: %w", err)
	}
	lengthAEAD := vmessAESGCM(vmessKDF16(respKey, vmessKDFSaltRespLengthKey))
	lengthIV := vmessKDF(respIV, vmessKDFSaltRespLengthIV)[:12]
	lengthPlain, err := lengthAEAD.Open(nil, lengthIV, encryptedLength[:], nil)
	if err != nil {
		return fmt.Errorf("decrypt vmess response header length: %w", err)
	}
	length := int(binary.BigEndian.Uint16(lengthPlain[:]))

	encryptedHeader := make([]byte, length+16)
	if _, err := io.ReadFull(s.Conn, encryptedHeader); err != nil {
		return fmt.Errorf("read vmess response header: %w", err)
	}
	headerAEAD := vmessAESGCM(vmessKDF16(respKey, vmessKDFSaltRespPayloadKey))
	headerIV := vmessKDF(respIV, vmessKDFSaltRespPayloadIV)[:12]
	header, err := headerAEAD.Open(nil, headerIV, encryptedHeader, nil)
	if err != nil {
		return fmt.Errorf("decrypt vmess response header: %w", err)
	}
	if len(header) < 2 || header[0] != s.respAuth {
		return errors.New("vmess response authentication mismatch")
	}
	s.respOpt = header[1]
	if s.respOpt&vmessOptChunkMasking != 0 {
		s.respMask = vmessShakeMask(respIV)
	}
	s.respOpened = true
	return nil
}

func (s *vmessSession) fill() error {
	if !s.respOpened {
		if err := s.openResponseHeader(); err != nil {
			return err
		}
	}
	if s.respOpt&vmessOptChunkStream == 0 {
		// Server agreed to stream without chunking: pass raw bytes through.
		buf := make([]byte, 16<<10)
		n, err := s.Conn.Read(buf)
		s.respBuf = buf[:n]
		return err
	}
	reqKey := sha256.Sum256(s.reqKey[:])
	reqIV := sha256.Sum256(s.reqIV[:])
	respKey := reqKey[:16]
	respIV := reqIV[:16]
	var payloadAEAD cipher.AEAD
	if s.cfg.Cipher == "chacha20-ietf-poly1305" {
		chacha, err := chacha20poly1305.New(vmessChaChaKey(respKey))
		if err != nil {
			return fmt.Errorf("init vmess response cipher: %w", err)
		}
		payloadAEAD = chacha
	} else {
		payloadAEAD = vmessAESGCM(respKey)
	}

	var wire [2]byte
	if _, err := io.ReadFull(s.Conn, wire[:]); err != nil {
		return fmt.Errorf("read vmess chunk length: %w", err)
	}
	size := binary.BigEndian.Uint16(wire[:])
	if s.respMask != nil {
		size ^= s.nextMask(s.respMask)
	}
	if size == 0 {
		return errors.New("invalid vmess chunk length")
	}
	tagged := make([]byte, int(size)+16)
	if _, err := io.ReadFull(s.Conn, tagged); err != nil {
		return fmt.Errorf("read vmess chunk payload: %w", err)
	}
	payload, err := payloadAEAD.Open(nil, vmessChunkNonce(respIV, s.respChunks), tagged, nil)
	if err != nil {
		return fmt.Errorf("decrypt vmess chunk payload: %w", err)
	}
	s.respChunks++
	s.respBuf = payload
	return nil
}

func (s *vmessSession) Read(p []byte) (int, error) {
	for len(s.respBuf) == 0 {
		if err := s.fill(); err != nil {
			return 0, err
		}
	}
	n := copy(p, s.respBuf)
	s.respBuf = s.respBuf[n:]
	return n, nil
}

// SerializeVMESSURL rebuilds a vmess:// base64(JSON) URI from a stored proxy
// record so the dialer can re-parse it without a second representation.
func SerializeVMESSURL(id, altID, host, port string, options map[string]string) string {
	option := func(name, fallback string) string {
		if v, ok := options[name]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return fallback
	}
	fields := map[string]string{
		"v":    "2",
		"ps":   "",
		"add":  host,
		"port": port,
		"id":   id,
		"aid":  altID,
		"scy":  option("scy", "auto"),
		"net":  option("net", "tcp"),
		"host": option("host", ""),
		"path": option("path", ""),
		"tls":  option("tls", ""),
		"sni":  option("sni", ""),
	}
	if raw := option("skip-cert-verify", ""); raw != "" {
		fields["skip-cert-verify"] = raw
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return "vmess://" + base64.RawURLEncoding.EncodeToString(encoded)
}
