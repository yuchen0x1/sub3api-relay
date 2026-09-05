package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/proxyutil"
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
	Name        string            `json:"name"`
	Protocol    string            `json:"protocol"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Username    string            `json:"username"`
	Password    string            `json:"password"`
	Options     map[string]string `json:"options,omitempty"`
	TLSInsecure bool              `json:"tls_insecure"`
}

func (h *ProxyHandler) ParseSubscription(c *gin.Context) {
	var req parseProxySubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	nodes, skipped, err := fetchAndParseProxySubscription(c.Request.Context(), req.URL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"proxies": nodes, "count": len(nodes), "skipped": skipped})
}

func fetchAndParseProxySubscription(ctx context.Context, rawURL string) ([]parsedProxySubscription, map[string]int, error) {
	cleaned := trimSubscriptionURLPunctuation(rawURL)
	normalized, err := validateSubscriptionURL(cleaned)
	if err != nil {
		return nil, nil, errors.New("subscription URL must be a public HTTPS URL")
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
			if _, err := validateSubscriptionURL(req.URL.String()); err != nil {
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
		return nil, nil, errors.New("invalid subscription request")
	}
	httpReq.Header.Set("Accept", "text/plain, application/octet-stream;q=0.9, */*;q=0.1")
	resp, err := fetchSubscriptionWithClientUA(client, httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, proxySubscriptionMaxBytes+1))
	if err != nil {
		return nil, nil, errors.New("failed to read subscription")
	}
	if len(body) > proxySubscriptionMaxBytes {
		return nil, nil, errors.New("subscription is too large")
	}
	return parseProxySubscriptionBody(body)
}

// 订阅商面板普遍按 User-Agent 分发内容：未知 UA 常被直接 404/403（实测
// xship 面板对自定义 UA 返回 404），clash 系 UA 返回 Clash YAML，v2ray 系
// UA 返回 base64 节点列表。两种格式解析器都已支持，因此先用 clash UA 请求，
// 被 403/404 拒绝时再用 v2rayN UA 重试一次。
const (
	proxySubscriptionPrimaryUA  = "clash-verge/1.6.6"
	proxySubscriptionFallbackUA = "v2rayN/6.45"
)

// fetchSubscriptionWithClientUA 以真实客户端 UA 拉取订阅，403/404 时换备用 UA 重试。
// 错误响应体被消费干净，避免连接无法复用。命中 Cloudflare 人机验证时给出
// 可操作的提示，因为任何程序化 UA 都无法通过该验证。
func fetchSubscriptionWithClientUA(client *http.Client, httpReq *http.Request) (*http.Response, error) {
	userAgents := []string{proxySubscriptionPrimaryUA, proxySubscriptionFallbackUA}
	var lastStatus int
	cfMitigated := false
	for _, ua := range userAgents {
		req := httpReq.Clone(httpReq.Context())
		req.Header.Set("User-Agent", ua)
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.New("failed to fetch subscription")
		}
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return resp, nil
		}
		lastStatus = resp.StatusCode
		if strings.EqualFold(strings.TrimSpace(resp.Header.Get("Cf-Mitigated")), "challenge") {
			cfMitigated = true
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
		if lastStatus != http.StatusForbidden && lastStatus != http.StatusNotFound {
			return nil, fmt.Errorf("subscription returned HTTP %d", lastStatus)
		}
	}
	if cfMitigated {
		return nil, fmt.Errorf(
			"subscription host requires Cloudflare human verification (HTTP %d), which automated fetching cannot pass; open the subscription URL in a browser and import the node links directly",
			lastStatus,
		)
	}
	return nil, fmt.Errorf("subscription returned HTTP %d", lastStatus)
}

// validateSubscriptionURL 校验订阅地址：格式合法、必须为 https、host 不得为
// localhost/私网字面量（解析后的 IP 由 safeSubscriptionDialContext 拦截）。
func validateSubscriptionURL(raw string) (string, error) {
	normalized, err := urlvalidator.ValidateURLFormat(raw, true)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalized)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return "", errors.New("subscription URL must be https")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", errors.New("subscription URL host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil &&
		(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
		return "", errors.New("subscription URL host is not allowed")
	}
	return normalized, nil
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

func parseProxySubscriptionBody(body []byte) ([]parsedProxySubscription, map[string]int, error) {
	text := strings.TrimSpace(strings.TrimPrefix(string(body), "\ufeff"))
	if text == "" {
		return nil, nil, errors.New("subscription is empty")
	}
	if containsSupportedProxyURI(text) {
		return parseProxySubscriptionLines(text)
	}
	if nodes, skipped, ok := parseClashProxyYAML(text); ok {
		return finishClashProxyYAML(nodes, skipped)
	}
	if decoded, err := decodeProxySubscriptionBase64(text); err == nil {
		// base64 载荷可能是 URI 列表，也可能是整个 Clash YAML 配置。
		if nodes, skipped, ok := parseClashProxyYAML(decoded); ok {
			return finishClashProxyYAML(nodes, skipped)
		}
		return parseProxySubscriptionLines(decoded)
	}
	return nil, nil, errors.New("subscription is not a supported proxy list")
}

// finishClashProxyYAML 汇总 Clash 路径的结果：无可用节点时给出带跳过原因的错误。
func finishClashProxyYAML(nodes []parsedProxySubscription, skipped map[string]int) ([]parsedProxySubscription, map[string]int, error) {
	if len(nodes) == 0 {
		detail := summarizeSkippedNodes(skipped)
		if detail != "" {
			return nil, skipped, fmt.Errorf("subscription contains no supported proxy nodes; skipped: %s", detail)
		}
		return nil, skipped, errors.New("subscription contains no supported proxy nodes")
	}
	return nodes, skipped, nil
}

// parseProxySubscriptionLines 逐行解析分享链接列表（URI 路径）。
func parseProxySubscriptionLines(text string) ([]parsedProxySubscription, map[string]int, error) {
	lines := strings.Fields(text)
	nodes := make([]parsedProxySubscription, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	skipped := make(map[string]int)
	for _, line := range lines {
		if len(nodes) >= proxySubscriptionMaxNodes {
			return nil, nil, fmt.Errorf("subscription contains more than %d nodes", proxySubscriptionMaxNodes)
		}
		node, err := parseProxySubscriptionURI(line)
		if err != nil {
			reason := err.Error()
			if reason == "" {
				reason = "unsupported node"
			}
			skipped[reason]++
			continue
		}
		optionsKey, _ := json.Marshal(node.Options)
		key := fmt.Sprintf("%s:%s:%d:%s:%s:%s:%t", node.Protocol, node.Host, node.Port, node.Username, node.Password, optionsKey, node.TLSInsecure)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		detail := summarizeSkippedNodes(skipped)
		if detail != "" {
			return nil, skipped, fmt.Errorf("subscription contains no supported proxy nodes; skipped: %s", detail)
		}
		return nil, skipped, errors.New("subscription contains no supported proxy nodes")
	}
	return nodes, skipped, nil
}

// summarizeSkippedNodes renders the per-reason skip counts so admins can see
// why parts of a subscription were not imported.
func summarizeSkippedNodes(skipped map[string]int) string {
	if len(skipped) == 0 {
		return ""
	}
	reasons := make([]string, 0, len(skipped))
	for reason, count := range skipped {
		reasons = append(reasons, fmt.Sprintf("%d %s", count, reason))
	}
	sort.Strings(reasons)
	return strings.Join(reasons, ", ")
}

// containsSupportedProxyURI reports whether the raw text already carries any
// proxy URI scheme the importer understands (otherwise it is base64).
func containsSupportedProxyURI(text string) bool {
	lower := strings.ToLower(text)
	for _, scheme := range []string{"vless://", "ss://", "trojan://", "vmess://"} {
		if strings.Contains(lower, scheme) {
			return true
		}
	}
	return false
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

func parseProxySubscriptionURI(raw string) (parsedProxySubscription, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return parsedProxySubscription{}, err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "vless":
		return parseVLESSSubscriptionURI(parsed)
	case "ss":
		return parseSSSubscriptionURI(parsed)
	case "trojan":
		return parseTrojanSubscriptionURI(parsed)
	case "vmess":
		return parseVMESSSubscriptionURI(raw)
	default:
		return parsedProxySubscription{}, fmt.Errorf("unsupported proxy scheme: %s", parsed.Scheme)
	}
}

// parseSSSubscriptionURI maps a SIP002 ss:// node onto the generic proxy
// record: the cipher travels in Username, the secret in Password.
func parseSSSubscriptionURI(parsed *url.URL) (parsedProxySubscription, error) {
	node, err := proxyutil.ParseSSNode(trimSubscriptionURLPunctuation(parsed.String()))
	if err != nil {
		return parsedProxySubscription{}, err
	}
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		name = net.JoinHostPort(node.Host, strconv.Itoa(node.Port))
	}
	return parsedProxySubscription{
		Name:     name,
		Protocol: "ss",
		Host:     node.Host,
		Port:     node.Port,
		Username: node.Method,
		Password: node.Password,
	}, nil
}

// parseTrojanSubscriptionURI maps a trojan:// node onto the generic proxy
// record: the password travels in Username, SNI/insecure in Options.
func parseTrojanSubscriptionURI(parsed *url.URL) (parsedProxySubscription, error) {
	node, err := proxyutil.ParseTrojanNode(trimSubscriptionURLPunctuation(parsed.String()))
	if err != nil {
		return parsedProxySubscription{}, err
	}
	options := map[string]string{}
	if node.SNI != "" && !strings.EqualFold(node.SNI, node.Host) {
		options["sni"] = node.SNI
	}
	if node.Insecure {
		options["allowInsecure"] = "true"
	}
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		name = net.JoinHostPort(node.Host, strconv.Itoa(node.Port))
	}
	return parsedProxySubscription{
		Name:     name,
		Protocol: "trojan",
		Host:     node.Host,
		Port:     node.Port,
		Username: node.Password,
		Options:  options,
	}, nil
}

// parseVMESSSubscriptionURI maps a vmess:// base64(JSON) node onto the
// generic proxy record: UUID in Username, alterId in Password, transport
// settings in Options.
func parseVMESSSubscriptionURI(raw string) (parsedProxySubscription, error) {
	node, err := proxyutil.ParseVMESSNode(trimSubscriptionURLPunctuation(raw))
	if err != nil {
		return parsedProxySubscription{}, err
	}
	options := map[string]string{
		"net": node.Network,
		"scy": node.Cipher,
	}
	if node.TLS {
		options["tls"] = "tls"
	}
	if node.Host != "" {
		options["host"] = node.Host
	}
	if node.Path != "" {
		options["path"] = node.Path
	}
	if node.SNI != "" {
		options["sni"] = node.SNI
	}
	if node.Insecure {
		options["skip-cert-verify"] = "true"
	}
	name := nodeDisplayName(raw, node.ServerHost, node.ServerPort)
	return parsedProxySubscription{
		Name:     name,
		Protocol: "vmess",
		Host:     node.ServerHost,
		Port:     node.ServerPort,
		Username: node.UUID,
		Password: "0",
		Options:  options,
	}, nil
}

// nodeDisplayName returns the URI fragment when present (vmess JSON "ps" is
// embedded in the fragment-less base64 payload, so fall back to host:port).
func nodeDisplayName(raw, host string, port int) string {
	if parsed, err := url.Parse(strings.TrimSpace(raw)); err == nil && strings.TrimSpace(parsed.Fragment) != "" {
		return strings.TrimSpace(parsed.Fragment)
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// parseVLESSSubscriptionURI accepts the VLESS transport used by common
// CDN-backed subscriptions. The gateway intentionally supports the portable
// TLS/WS form only; Reality, XTLS flow and other transports require a full
// Xray/Sing-box runtime and cannot be represented by an HTTP dialer.
func parseVLESSSubscriptionURI(parsed *url.URL) (parsedProxySubscription, error) {
	if parsed == nil || parsed.User == nil {
		return parsedProxySubscription{}, errors.New("VLESS proxy is missing UUID")
	}
	if _, ok := parsed.User.Password(); ok {
		return parsedProxySubscription{}, errors.New("VLESS password authentication is not supported")
	}
	uuidValue, err := uuid.Parse(parsed.User.Username())
	if err != nil {
		return parsedProxySubscription{}, errors.New("VLESS UUID is invalid")
	}
	host := parsed.Hostname()
	if host == "" {
		return parsedProxySubscription{}, errors.New("VLESS proxy is missing host")
	}
	port := 443
	if rawPort := parsed.Port(); rawPort != "" {
		port, err = strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return parsedProxySubscription{}, errors.New("invalid VLESS proxy port")
		}
	}
	options, err := normalizeVLESSSubscriptionOptions(parsed.Query(), host)
	if err != nil {
		return parsedProxySubscription{}, err
	}
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		name = net.JoinHostPort(host, strconv.Itoa(port))
	}
	return parsedProxySubscription{
		Name:     name,
		Protocol: "vless",
		Host:     host,
		Port:     port,
		Username: uuidValue.String(),
		Options:  options,
	}, nil
}

func normalizeVLESSSubscriptionOptions(query url.Values, serverHost string) (map[string]string, error) {
	// Unknown keys are ignored: real-world subscription generators append
	// transport-specific params (serviceName, quicSecurity, headerType, sid,
	// pbk, ...) to every node, and rejecting them wholesale used to discard
	// otherwise importable WS/TLS nodes. Only values this dialer cannot
	// honor are rejected below.
	security := strings.ToLower(strings.TrimSpace(query.Get("security")))
	if security == "" {
		security = "tls"
	}
	if security != "tls" && security != "none" {
		return nil, fmt.Errorf("unsupported VLESS security: %s", security)
	}
	transportType := strings.ToLower(strings.TrimSpace(query.Get("type")))
	if transportType == "" {
		transportType = "ws"
	}
	if transportType != "ws" {
		return nil, fmt.Errorf("unsupported VLESS transport: %s", transportType)
	}
	encryption := strings.ToLower(strings.TrimSpace(query.Get("encryption")))
	if encryption == "" {
		encryption = "none"
	}
	if encryption != "none" {
		return nil, fmt.Errorf("unsupported VLESS encryption: %s", encryption)
	}
	if flow := strings.TrimSpace(query.Get("flow")); flow != "" {
		return nil, fmt.Errorf("unsupported VLESS flow: %s", flow)
	}
	if headerType := strings.ToLower(strings.TrimSpace(query.Get("headerType"))); headerType != "" && headerType != "none" {
		return nil, fmt.Errorf("unsupported VLESS header type: %s", headerType)
	}
	path := query.Get("path")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") {
		return nil, errors.New("invalid VLESS WebSocket path")
	}
	sni := strings.TrimSpace(query.Get("sni"))
	if sni == "" {
		sni = serverHost
	}
	hostHeader := strings.TrimSpace(query.Get("host"))
	if hostHeader == "" {
		hostHeader = sni
	}
	options := map[string]string{
		"security":   security,
		"type":       transportType,
		"encryption": encryption,
		"sni":        sni,
		"host":       hostHeader,
		"path":       path,
	}
	for _, key := range []string{"mode", "fp", "alpn"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			options[key] = value
		}
	}
	if rawInsecure := strings.TrimSpace(query.Get("insecure")); rawInsecure != "" {
		insecure, err := strconv.ParseBool(rawInsecure)
		if err != nil {
			return nil, errors.New("invalid VLESS insecure option")
		}
		if insecure {
			options["insecure"] = "true"
		}
	}
	return options, nil
}
