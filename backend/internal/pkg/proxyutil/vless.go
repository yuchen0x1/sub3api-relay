package proxyutil

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/google/uuid"
)

const (
	vlessVersion       = byte(0)
	vlessCommandTCP    = byte(1)
	vlessAddressIPv4   = byte(1)
	vlessAddressDomain = byte(2)
	vlessAddressIPv6   = byte(3)
	vlessDefaultPort   = "443"
)

// vlessConfig describes the VLESS transport that can be represented by a
// net/http DialContext. The gateway deliberately supports the widely used
// VLESS-over-TLS-WebSocket form (plus plaintext WebSocket), while rejecting
// Reality/XTLS and other transports that need a full Xray runtime.
type vlessConfig struct {
	UUID       uuid.UUID
	ServerHost string
	ServerPort string
	Security   string
	SNI        string
	Host       string
	Path       string
	Insecure   bool
}

func newVLESSDialContext(proxyURL *url.URL) (func(context.Context, string, string) (net.Conn, error), error) {
	cfg, err := parseVLESSConfig(proxyURL)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, network, destination string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("vless only supports TCP dialing, got %s", network)
		}
		host, port, err := net.SplitHostPort(destination)
		if err != nil {
			return nil, fmt.Errorf("invalid VLESS destination: %w", err)
		}
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return nil, errors.New("invalid VLESS destination port")
		}

		wsScheme := "wss"
		if cfg.Security == "none" {
			wsScheme = "ws"
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
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   true,
			ForceAttemptHTTP2:   false,
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     10 * time.Second,
		}
		if cfg.Security == "tls" {
			transport.TLSClientConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				ServerName:         cfg.SNI,
				InsecureSkipVerify: cfg.Insecure, // opt-in is carried by the subscription itself
			}
		}
		httpClient := &http.Client{Transport: transport}
		wsConn, resp, err := coderws.Dial(ctx, wsURL.String(), &coderws.DialOptions{
			HTTPClient: httpClient,
			Host:       cfg.Host,
			HTTPHeader: http.Header{"User-Agent": []string{"sub3api-vless/1.0"}},
		})
		if err != nil {
			if resp != nil {
				return nil, fmt.Errorf("VLESS WebSocket handshake failed with HTTP %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("VLESS WebSocket handshake failed: %w", err)
		}
		// The request context bounds only the handshake. The resulting stream
		// must survive after net/http returns from DialContext and is closed by
		// the transport when the connection is retired.
		stream := coderws.NetConn(context.Background(), wsConn, coderws.MessageBinary)
		header, err := buildVLESSRequestHeader(cfg.UUID, host, portNumber)
		if err != nil {
			_ = stream.Close()
			return nil, err
		}
		if _, err := stream.Write(header); err != nil {
			_ = stream.Close()
			return nil, fmt.Errorf("write VLESS request header: %w", err)
		}
		return &vlessConn{Conn: stream}, nil
	}, nil
}

func parseVLESSConfig(proxyURL *url.URL) (vlessConfig, error) {
	if proxyURL == nil || !strings.EqualFold(proxyURL.Scheme, "vless") {
		return vlessConfig{}, errors.New("invalid VLESS proxy URL")
	}
	if proxyURL.User == nil {
		return vlessConfig{}, errors.New("VLESS proxy missing UUID")
	}
	if _, ok := proxyURL.User.Password(); ok {
		return vlessConfig{}, errors.New("VLESS password authentication is not supported")
	}
	id, err := uuid.Parse(proxyURL.User.Username())
	if err != nil {
		return vlessConfig{}, errors.New("VLESS UUID is invalid")
	}
	serverHost := proxyURL.Hostname()
	if serverHost == "" {
		return vlessConfig{}, errors.New("VLESS proxy missing host")
	}
	serverPort := proxyURL.Port()
	if serverPort == "" {
		serverPort = vlessDefaultPort
	}
	if port, err := strconv.Atoi(serverPort); err != nil || port < 1 || port > 65535 {
		return vlessConfig{}, errors.New("invalid VLESS proxy port")
	}
	query := proxyURL.Query()
	security := strings.ToLower(strings.TrimSpace(query.Get("security")))
	if security == "" {
		security = "tls"
	}
	if security != "tls" && security != "none" {
		return vlessConfig{}, fmt.Errorf("unsupported VLESS security: %s", security)
	}
	transportType := strings.ToLower(strings.TrimSpace(query.Get("type")))
	if transportType == "" {
		transportType = "ws"
	}
	if transportType != "ws" {
		return vlessConfig{}, fmt.Errorf("unsupported VLESS transport: %s", transportType)
	}
	encryption := strings.ToLower(strings.TrimSpace(query.Get("encryption")))
	if encryption != "" && encryption != "none" {
		return vlessConfig{}, fmt.Errorf("unsupported VLESS encryption: %s", encryption)
	}
	if flow := strings.TrimSpace(query.Get("flow")); flow != "" {
		return vlessConfig{}, fmt.Errorf("unsupported VLESS flow: %s", flow)
	}
	path := query.Get("path")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") {
		return vlessConfig{}, errors.New("invalid VLESS WebSocket path")
	}
	sni := strings.TrimSpace(query.Get("sni"))
	if sni == "" {
		sni = serverHost
	}
	if err := validateVLESSHeaderValue(sni); err != nil {
		return vlessConfig{}, fmt.Errorf("invalid VLESS SNI: %w", err)
	}
	hostHeader := strings.TrimSpace(query.Get("host"))
	if hostHeader == "" {
		hostHeader = sni
	}
	if err := validateVLESSHeaderValue(hostHeader); err != nil {
		return vlessConfig{}, fmt.Errorf("invalid VLESS WebSocket host: %w", err)
	}
	insecure := false
	if raw := strings.TrimSpace(query.Get("insecure")); raw != "" {
		insecure, err = strconv.ParseBool(raw)
		if err != nil {
			return vlessConfig{}, errors.New("invalid VLESS insecure option")
		}
	}
	return vlessConfig{
		UUID:       id,
		ServerHost: serverHost,
		ServerPort: serverPort,
		Security:   security,
		SNI:        sni,
		Host:       hostHeader,
		Path:       path,
		Insecure:   insecure,
	}, nil
}

func validateVLESSHeaderValue(value string) error {
	if value == "" || len(value) > 255 || strings.ContainsAny(value, "\r\n") {
		return errors.New("header value is invalid")
	}
	return nil
}

func buildVLESSRequestHeader(id uuid.UUID, host string, port int) ([]byte, error) {
	if id == uuid.Nil {
		return nil, errors.New("VLESS UUID is empty")
	}
	if host == "" || port < 1 || port > 65535 {
		return nil, errors.New("invalid VLESS destination")
	}
	header := make([]byte, 0, 24+len(host))
	header = append(header, vlessVersion)
	header = append(header, id[:]...)
	header = append(header, 0) // no request addons
	header = append(header, vlessCommandTCP)
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], uint16(port))
	header = append(header, portBytes[:]...)
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			header = append(header, vlessAddressIPv4)
			return append(header, ip4...), nil
		}
		if ip16 := ip.To16(); ip16 != nil {
			header = append(header, vlessAddressIPv6)
			return append(header, ip16...), nil
		}
	}
	if len(host) > 255 || strings.ContainsAny(host, "\x00\r\n") {
		return nil, errors.New("invalid VLESS destination host")
	}
	header = append(header, vlessAddressDomain, byte(len(host)))
	return append(header, host...), nil
}

// vlessConn strips the VLESS response header once, then exposes the raw TCP
// stream to net/http. coder/websocket's NetConn provides correct WebSocket
// message framing and deadline handling underneath.
type vlessConn struct {
	net.Conn
	readerOnce sync.Once
	reader     *bufio.Reader
	headerErr  error
}

func (c *vlessConn) ensureResponseHeader() error {
	c.readerOnce.Do(func() {
		c.reader = bufio.NewReader(c.Conn)
		var header [2]byte
		if _, err := io.ReadFull(c.reader, header[:]); err != nil {
			c.headerErr = fmt.Errorf("read VLESS response header: %w", err)
			return
		}
		if header[0] != vlessVersion {
			c.headerErr = fmt.Errorf("unexpected VLESS response version %d", header[0])
			return
		}
		addonLength := int(header[1])
		if addonLength > 0 {
			addons := make([]byte, addonLength)
			if _, err := io.ReadFull(c.reader, addons); err != nil {
				c.headerErr = fmt.Errorf("read VLESS response addons: %w", err)
			}
		}
	})
	return c.headerErr
}

func (c *vlessConn) Read(p []byte) (int, error) {
	if err := c.ensureResponseHeader(); err != nil {
		return 0, err
	}
	return c.reader.Read(p)
}
