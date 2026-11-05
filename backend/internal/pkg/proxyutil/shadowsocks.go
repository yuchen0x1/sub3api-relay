package proxyutil

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // Shadowsocks SIP004 mandates the MD5-based EVP_BytesToKey KDF
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// Shadowsocks (SIP004 AEAD) support. The gateway implements the TCP-only
// modern AEAD ciphers that current subscriptions use; stream ciphers and the
// 2022 blake3 ciphers are rejected at parse time with an explicit reason.

const (
	ssSubkeyInfo      = "ss-subkey"
	ssChunkMaxPayload = 0x3FFF
	ssMaxTagSize      = 16
	ssHandshakeBound  = 10 * time.Second
)

// ssCipher describes one supported AEAD method.
type ssCipher struct {
	keyLen  int
	newAEAD func(key []byte) (cipher.AEAD, error)
}

var ssCiphers = map[string]ssCipher{
	"aes-128-gcm":            {keyLen: 16, newAEAD: newSSAesGCM},
	"aes-256-gcm":            {keyLen: 32, newAEAD: newSSAesGCM},
	"chacha20-ietf-poly1305": {keyLen: 32, newAEAD: chacha20poly1305.New},
}

func newSSAesGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// IsSupportedSSMethod reports whether a subscription method can be dialed.
func IsSupportedSSMethod(method string) bool {
	_, ok := ssCiphers[strings.ToLower(strings.TrimSpace(method))]
	return ok
}

type ssConfig struct {
	Method   string
	Password string
	Host     string
	Port     string
}

// parseSSURI validates an ss:// URI string and returns its config.
func parseSSURI(raw string) (ssConfig, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ssConfig{}, errors.New("invalid shadowsocks proxy URL")
	}
	return parseSSConfig(parsed)
}

// parseSSConfig parses a SIP002 ss:// URL. The userinfo is either
// websafe-base64 of "method:password" or a literal "method:password" pair.
func parseSSConfig(proxyURL *url.URL) (ssConfig, error) {
	if proxyURL == nil || !strings.EqualFold(proxyURL.Scheme, "ss") {
		return ssConfig{}, errors.New("invalid shadowsocks proxy URL")
	}
	if proxyURL.User == nil || proxyURL.User.Username() == "" {
		return ssConfig{}, errors.New("shadowsocks proxy missing method:password userinfo")
	}
	method, password, err := decodeSSUserInfo(proxyURL.User)
	if err != nil {
		return ssConfig{}, err
	}
	method = strings.ToLower(strings.TrimSpace(method))
	if _, ok := ssCiphers[method]; !ok {
		if strings.Contains(method, "2022") {
			return ssConfig{}, fmt.Errorf("unsupported shadowsocks method %q (2022 ciphers are not supported)", method)
		}
		return ssConfig{}, fmt.Errorf("unsupported shadowsocks method %q (supported: aes-128-gcm, aes-256-gcm, chacha20-ietf-poly1305)", method)
	}
	host := proxyURL.Hostname()
	if host == "" {
		return ssConfig{}, errors.New("shadowsocks proxy missing host")
	}
	port := proxyURL.Port()
	if port == "" {
		port = "443"
	}
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
		return ssConfig{}, errors.New("invalid shadowsocks proxy port")
	}
	if proxyURL.Query().Get("plugin") != "" {
		return ssConfig{}, errors.New("shadowsocks plugins are not supported")
	}
	return ssConfig{Method: method, Password: password, Host: host, Port: port}, nil
}

func decodeSSUserInfo(user *url.Userinfo) (method, password string, err error) {
	raw := user.Username()
	if decoded, ok := decodeSSBase64Userinfo(raw); ok {
		if m, p, found := strings.Cut(decoded, ":"); found && m != "" {
			if unescaped, err := url.QueryUnescape(p); err == nil {
				p = unescaped
			}
			return m, p, nil
		}
		return "", "", errors.New("shadowsocks userinfo is invalid")
	}
	method, err = url.QueryUnescape(raw)
	if err != nil {
		return "", "", errors.New("shadowsocks method is invalid")
	}
	userPassword, hasPassword := user.Password()
	if !hasPassword {
		return "", "", errors.New("shadowsocks userinfo must be method:password")
	}
	if password, err = url.QueryUnescape(userPassword); err != nil {
		return "", "", errors.New("shadowsocks password is invalid")
	}
	return method, password, nil
}

func decodeSSBase64Userinfo(raw string) (string, bool) {
	if strings.Contains(raw, ":") {
		return "", false
	}
	compact := strings.NewReplacer("+", "-", "/", "_").Replace(raw)
	trimmed := strings.TrimRight(compact, "=")
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.RawStdEncoding} {
		decoded, err := enc.DecodeString(trimmed)
		if err == nil && utf8.Valid(decoded) && strings.Contains(string(decoded), ":") {
			return string(decoded), true
		}
	}
	return "", false
}

// ssKDF derives the master key with the SIP004 EVP_BytesToKey(MD5) schedule.
func ssKDF(password string, keyLen int) []byte {
	var out, prev []byte
	for len(out) < keyLen {
		h := md5.New()
		_, _ = h.Write(prev)
		_, _ = h.Write([]byte(password))
		prev = h.Sum(prev[:0])
		out = append(out, prev...)
	}
	return out[:keyLen]
}

func newSSConn(conn net.Conn, cfg ssConfig, destination string) (net.Conn, error) {
	cipherSpec := ssCiphers[cfg.Method]
	master := ssKDF(cfg.Password, cipherSpec.keyLen)

	salt := make([]byte, cipherSpec.keyLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate shadowsocks salt: %w", err)
	}
	writeAEAD, err := cipherSpec.newAEAD(ssSubkey(master, salt, cipherSpec.keyLen))
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(salt); err != nil {
		return nil, fmt.Errorf("write shadowsocks salt: %w", err)
	}

	target, err := buildSSTargetAddress(destination)
	if err != nil {
		return nil, err
	}

	writeKey := &ssStreamKey{aead: writeAEAD}
	if err := writeKey.writeChunk(conn, target); err != nil {
		return nil, err
	}
	// The server's salt is the first bytes of its reply and only arrives
	// with the response, so the read key is derived lazily in fill().
	return &ssConn{Conn: conn, writeKey: writeKey, cipherSpec: cipherSpec, master: master}, nil
}

// newSSDialContext returns a DialContext that tunnels TCP through a
// Shadowsocks AEAD server.
func newSSDialContext(proxyURL *url.URL) (func(context.Context, string, string) (net.Conn, error), error) {
	cfg, err := parseSSConfig(proxyURL)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, network, destination string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("shadowsocks only supports TCP dialing, got %s", network)
		}
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, cfg.Port))
		if err != nil {
			return nil, fmt.Errorf("connect shadowsocks server: %w", err)
		}
		handshakeDeadline := time.Now().Add(ssHandshakeBound)
		if deadline, ok := ctx.Deadline(); ok && deadline.Before(handshakeDeadline) {
			handshakeDeadline = deadline
		}
		_ = conn.SetDeadline(handshakeDeadline)
		tunneled, err := newSSConn(conn, cfg, destination)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		// Handshake done; the stream must not inherit the handshake deadline.
		_ = tunneled.SetDeadline(time.Time{})
		return tunneled, nil
	}, nil
}

func ssSubkey(master, salt []byte, keyLen int) []byte {
	subkey := make([]byte, keyLen)
	kdf := hkdf.New(sha1.New, master, salt, []byte(ssSubkeyInfo))
	if _, err := io.ReadFull(kdf, subkey); err != nil {
		// hkdf with a bounded output only fails on crypto-level misuse.
		panic("shadowsocks hkdf failed: " + err.Error())
	}
	return subkey
}

func buildSSTargetAddress(destination string) ([]byte, error) {
	host, port, err := net.SplitHostPort(destination)
	if err != nil {
		return nil, fmt.Errorf("invalid shadowsocks destination: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, errors.New("invalid shadowsocks destination port")
	}
	var buf []byte
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, 1)
			buf = append(buf, ip4...)
		} else {
			buf = append(buf, 4)
			buf = append(buf, ip.To16()...)
		}
	} else {
		if len(host) > 255 || strings.ContainsAny(host, "\x00\r\n") {
			return nil, errors.New("invalid shadowsocks destination host")
		}
		buf = append(buf, 3, byte(len(host)))
		buf = append(buf, host...)
	}
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], uint16(portNumber))
	return append(buf, portBytes[:]...), nil
}

// ssStreamKey carries one direction's AEAD with its incrementing nonce.
type ssStreamKey struct {
	aead   cipher.AEAD
	nonce  [12]byte
	inited bool
}

func (k *ssStreamKey) nextNonce() []byte {
	if !k.inited {
		k.inited = true
		return k.nonce[:]
	}
	incrementLE(k.nonce[:])
	return k.nonce[:]
}

func (k *ssStreamKey) writeChunk(w io.Writer, payload []byte) error {
	var header [2 + ssMaxTagSize]byte
	binary.BigEndian.PutUint16(header[:2], uint16(len(payload)))
	sealed := k.aead.Seal(header[:0], k.nextNonce(), header[:2], nil)
	if _, err := w.Write(sealed); err != nil {
		return fmt.Errorf("write shadowsocks chunk length: %w", err)
	}
	body := k.aead.Seal(nil, k.nextNonce(), payload, nil)
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write shadowsocks chunk payload: %w", err)
	}
	return nil
}

// ssConn turns the chunked AEAD stream into a plain net.Conn for net/http.
type ssConn struct {
	net.Conn
	writeKey   *ssStreamKey
	cipherSpec ssCipher
	master     []byte

	readKey  *ssStreamKey
	saltRead bool
	readBuf  []byte
}

func (c *ssConn) Write(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		end := total + ssChunkMaxPayload
		if end > len(p) {
			end = len(p)
		}
		if err := c.writeKey.writeChunk(c.Conn, p[total:end]); err != nil {
			return total, err
		}
		total = end
	}
	return total, nil
}

func (c *ssConn) fill() error {
	if !c.saltRead {
		readSalt := make([]byte, c.cipherSpec.keyLen)
		if _, err := io.ReadFull(c.Conn, readSalt); err != nil {
			return fmt.Errorf("read shadowsocks salt: %w", err)
		}
		readAEAD, err := c.cipherSpec.newAEAD(ssSubkey(c.master, readSalt, c.cipherSpec.keyLen))
		if err != nil {
			return err
		}
		c.readKey = &ssStreamKey{aead: readAEAD}
		c.saltRead = true
	}
	var lengthBlock [2 + ssMaxTagSize]byte
	if _, err := io.ReadFull(c.Conn, lengthBlock[:]); err != nil {
		return fmt.Errorf("read shadowsocks chunk length: %w", err)
	}
	plain, err := c.readKey.aead.Open(nil, c.readKey.nextNonce(), lengthBlock[:], nil)
	if err != nil {
		return fmt.Errorf("decrypt shadowsocks chunk length: %w", err)
	}
	length := int(binary.BigEndian.Uint16(plain))
	if length == 0 || length > ssChunkMaxPayload {
		return errors.New("invalid shadowsocks chunk length")
	}
	tagged := make([]byte, length+ssMaxTagSize)
	if _, err := io.ReadFull(c.Conn, tagged); err != nil {
		return fmt.Errorf("read shadowsocks chunk payload: %w", err)
	}
	payload, err := c.readKey.aead.Open(nil, c.readKey.nextNonce(), tagged, nil)
	if err != nil {
		return fmt.Errorf("decrypt shadowsocks chunk payload: %w", err)
	}
	c.readBuf = payload
	return nil
}

func (c *ssConn) Read(p []byte) (int, error) {
	for len(c.readBuf) == 0 {
		if err := c.fill(); err != nil {
			return 0, err
		}
	}
	n := copy(p, c.readBuf)
	c.readBuf = c.readBuf[n:]
	return n, nil
}

// incrementLE advances a little-endian counter in place (SIP004 nonce rule).
func incrementLE(b []byte) {
	for i := range b {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}
