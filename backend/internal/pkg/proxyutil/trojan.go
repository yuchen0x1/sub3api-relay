package proxyutil

import (
	"context"
	"crypto/sha256" //nolint:gosec // Trojan's fixed protocol uses SHA224 of the password as auth
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Trojan support: password auth over TLS followed by a SOCKS5-style target
// address, as specified by the trojan-gfw protocol (trojan://password@host).

const trojanHandshakeBound = 10 * time.Second

type trojanConfig struct {
	Password string
	Host     string
	Port     string
	SNI      string
	Insecure bool
}

// parseTrojanURI validates a trojan:// URI string and returns its config.
func parseTrojanURI(raw string) (trojanConfig, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return trojanConfig{}, errors.New("invalid trojan proxy URL")
	}
	return parseTrojanConfig(parsed)
}

// parseTrojanConfig parses a trojan:// URL. The userinfo username carries the
// password (the URI scheme has no separate username).
func parseTrojanConfig(proxyURL *url.URL) (trojanConfig, error) {
	if proxyURL == nil || !strings.EqualFold(proxyURL.Scheme, "trojan") {
		return trojanConfig{}, errors.New("invalid trojan proxy URL")
	}
	password := ""
	if proxyURL.User != nil {
		password = proxyURL.User.Username()
		if encoded, err := url.QueryUnescape(password); err == nil {
			password = encoded
		}
		if userPassword, hasPassword := proxyURL.User.Password(); hasPassword && userPassword != "" {
			return trojanConfig{}, errors.New("trojan proxy takes a single password credential")
		}
	}
	if strings.TrimSpace(password) == "" {
		return trojanConfig{}, errors.New("trojan proxy missing password")
	}
	host := proxyURL.Hostname()
	if host == "" {
		return trojanConfig{}, errors.New("trojan proxy missing host")
	}
	port := proxyURL.Port()
	if port == "" {
		port = "443"
	}
	if number, err := strconv.Atoi(port); err != nil || number < 1 || number > 65535 {
		return trojanConfig{}, errors.New("invalid trojan proxy port")
	}
	query := proxyURL.Query()
	sni := strings.TrimSpace(query.Get("sni"))
	if sni == "" {
		sni = strings.TrimSpace(query.Get("peer"))
	}
	if sni == "" {
		sni = host
	}
	insecure := false
	for _, key := range []string{"allowInsecure", "insecure"} {
		if raw := strings.TrimSpace(query.Get(key)); raw != "" {
			parsed, err := strconv.ParseBool(raw)
			if err != nil {
				return trojanConfig{}, errors.New("invalid trojan insecure option")
			}
			insecure = insecure || parsed
		}
	}
	return trojanConfig{Password: password, Host: host, Port: port, SNI: sni, Insecure: insecure}, nil
}

// newTrojanDialContext returns a DialContext that tunnels TCP through a
// Trojan TLS server.
func newTrojanDialContext(proxyURL *url.URL) (func(context.Context, string, string) (net.Conn, error), error) {
	cfg, err := parseTrojanConfig(proxyURL)
	if err != nil {
		return nil, err
	}
	authHash := trojanAuthHash(cfg.Password)
	return func(ctx context.Context, network, destination string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("trojan only supports TCP dialing, got %s", network)
		}
		host, port, err := net.SplitHostPort(destination)
		if err != nil {
			return nil, fmt.Errorf("invalid trojan destination: %w", err)
		}
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return nil, errors.New("invalid trojan destination port")
		}

		var dialer net.Dialer
		rawConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, cfg.Port))
		if err != nil {
			return nil, fmt.Errorf("connect trojan server: %w", err)
		}
		tlsConn := tls.Client(rawConn, &tls.Config{
			ServerName:         cfg.SNI,
			InsecureSkipVerify: cfg.Insecure, //nolint:gosec // opt-in is carried by the subscription itself
			MinVersion:         tls.VersionTLS12,
		})
		handshakeCtx, cancel := context.WithTimeout(ctx, trojanHandshakeBound)
		defer cancel()
		if err := tlsConn.HandshakeContext(handshakeCtx); err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("trojan TLS handshake failed: %w", err)
		}

		trojanConn := &trojanConn{Conn: tlsConn}
		if err := trojanConn.writeRequest(authHash, host, portNumber); err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		// The handshake deadline must not clip the tunneled stream; trojan
		// sends no response for TCP requests, so there is nothing left to
		// read before handing the connection to net/http.
		_ = trojanConn.SetDeadline(time.Time{})
		return trojanConn, nil
	}, nil
}

func trojanAuthHash(password string) []byte {
	sum := sha256.Sum224([]byte(password))
	dst := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(dst, sum[:])
	return dst
}

type trojanConn struct {
	net.Conn
}

// writeRequest sends hex(SHA224(password)) CRLF socks5-address CRLF; after
// this the connection is a raw tunnel to the destination.
func (c *trojanConn) writeRequest(authHash []byte, host string, port int) error {
	var request []byte
	request = append(request, authHash...)
	request = append(request, '\r', '\n')
	if err := appendTrojanAddress(&request, host, port); err != nil {
		return err
	}
	request = append(request, '\r', '\n')
	if _, err := c.Write(request); err != nil {
		return fmt.Errorf("write trojan request: %w", err)
	}
	return nil
}

// appendTrojanAddress appends ATYP + address + port (port last, big endian),
// the SOCKS5 address layout Trojan shares with Shadowsocks.
func appendTrojanAddress(dst *[]byte, host string, port int) error {
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			*dst = append(*dst, 1)
			*dst = append(*dst, ip4...)
		} else {
			*dst = append(*dst, 4)
			*dst = append(*dst, ip.To16()...)
		}
	} else {
		if len(host) > 255 || strings.ContainsAny(host, "\x00\r\n") {
			return errors.New("invalid trojan destination host")
		}
		*dst = append(*dst, 3, byte(len(host)))
		*dst = append(*dst, host...)
	}
	var portBytes [2]byte
	portBytes[0] = byte(port >> 8)
	portBytes[1] = byte(port)
	*dst = append(*dst, portBytes[:]...)
	return nil
}
