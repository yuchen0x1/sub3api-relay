package admin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/util/urlvalidator"
)

const (
	proxySubscriptionMaxBytes = 2 << 20
	proxySubscriptionMaxNodes = 500
	proxySubscriptionTimeout  = 15 * time.Second
)

type parseProxySubscriptionRequest struct {
	URL string `json:"url" binding:"required"`
}

type parsedProxySubscription struct {
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	TLSInsecure bool   `json:"tls_insecure"`
}

func (h *ProxyHandler) ParseSubscription(c *gin.Context) {
	var req parseProxySubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	nodes, err := fetchAndParseProxySubscription(c.Request.Context(), req.URL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"proxies": nodes, "count": len(nodes)})
}

func fetchAndParseProxySubscription(ctx context.Context, rawURL string) ([]parsedProxySubscription, error) {
	cleaned := trimSubscriptionURLPunctuation(rawURL)
	normalized, err := urlvalidator.ValidateHTTPSURL(cleaned, urlvalidator.ValidationOptions{})
	if err != nil {
		return nil, errors.New("subscription URL must be a public HTTPS URL")
	}
	client := &http.Client{
		Timeout: proxySubscriptionTimeout,
		Transport: &http.Transport{
			Proxy:       nil,
			DialContext: safeSubscriptionDialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many subscription redirects")
			}
			if _, err := urlvalidator.ValidateHTTPSURL(req.URL.String(), urlvalidator.ValidationOptions{}); err != nil {
				return errors.New("subscription redirect is not allowed")
			}
			if len(via) == 0 || !sameSubscriptionOrigin(via[0].URL, req.URL) {
				return errors.New("subscription redirect must remain on the same origin")
			}
			req.Header.Del("Referer")
			return nil
		},
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return nil, errors.New("invalid subscription request")
	}
	httpReq.Header.Set("Accept", "text/plain, application/octet-stream;q=0.9, */*;q=0.1")
	httpReq.Header.Set("User-Agent", "sub3api-proxy-subscription/1.0")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.New("failed to fetch subscription")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, proxySubscriptionMaxBytes+1))
	if err != nil {
		return nil, errors.New("failed to read subscription")
	}
	if len(body) > proxySubscriptionMaxBytes {
		return nil, errors.New("subscription is too large")
	}
	return parseProxySubscriptionBody(body)
}

func sameSubscriptionOrigin(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

var blockedSubscriptionPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
}

func safeSubscriptionDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid subscription host")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("failed to resolve subscription host")
	}
	for _, ip := range ips {
		if isBlockedSubscriptionIP(ip) {
			return nil, errors.New("subscription host resolved to a private address")
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

func isBlockedSubscriptionIP(ip net.IP) bool {
	address, err := netip.ParseAddr(ip.String())
	if err != nil {
		return true
	}
	address = address.Unmap()
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() ||
		address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() || address.Is4In6() {
		return true
	}
	for _, prefix := range blockedSubscriptionPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func trimSubscriptionURLPunctuation(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "，。；、！!？?")
}

func parseProxySubscriptionBody(body []byte) ([]parsedProxySubscription, error) {
	text := strings.TrimSpace(strings.TrimPrefix(string(body), "\ufeff"))
	if text == "" {
		return nil, errors.New("subscription is empty")
	}
	if !strings.Contains(strings.ToLower(text), "hysteria2://") && !strings.Contains(strings.ToLower(text), "hy2://") {
		decoded, err := decodeProxySubscriptionBase64(text)
		if err != nil {
			return nil, errors.New("subscription is not a supported proxy list")
		}
		text = decoded
	}
	lines := strings.Fields(text)
	nodes := make([]parsedProxySubscription, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if len(nodes) >= proxySubscriptionMaxNodes {
			return nil, fmt.Errorf("subscription contains more than %d nodes", proxySubscriptionMaxNodes)
		}
		node, err := parseHysteria2SubscriptionURI(line)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s:%t", node.Host, node.Port, node.Password, node.TLSInsecure)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, errors.New("subscription contains no supported Hysteria2 nodes")
	}
	return nodes, nil
}

func decodeProxySubscriptionBase64(raw string) (string, error) {
	compact := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "").Replace(raw)
	encodings := []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(compact)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), nil
		}
	}
	return "", errors.New("invalid base64 subscription")
}

func parseHysteria2SubscriptionURI(raw string) (parsedProxySubscription, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return parsedProxySubscription{}, err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "hysteria2" && scheme != "hy2" {
		return parsedProxySubscription{}, errors.New("unsupported proxy scheme")
	}
	host := parsed.Hostname()
	if host == "" {
		return parsedProxySubscription{}, errors.New("missing proxy host")
	}
	port := 443
	if rawPort := parsed.Port(); rawPort != "" {
		port, err = strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return parsedProxySubscription{}, errors.New("invalid proxy port")
		}
	}
	auth := ""
	if parsed.User != nil {
		auth = parsed.User.Username()
		if password, ok := parsed.User.Password(); ok {
			auth += ":" + password
		}
	}
	query := parsed.Query()
	for key := range query {
		if key != "insecure" {
			return parsedProxySubscription{}, fmt.Errorf("unsupported Hysteria2 option: %s", key)
		}
	}
	insecure := false
	if rawInsecure := strings.TrimSpace(query.Get("insecure")); rawInsecure != "" {
		insecure, err = strconv.ParseBool(rawInsecure)
		if err != nil {
			return parsedProxySubscription{}, errors.New("invalid Hysteria2 insecure option")
		}
	}
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		name = net.JoinHostPort(host, strconv.Itoa(port))
	}
	return parsedProxySubscription{
		Name:        name,
		Protocol:    "hysteria2",
		Host:        host,
		Port:        port,
		Password:    auth,
		TLSInsecure: insecure,
	}, nil
}
