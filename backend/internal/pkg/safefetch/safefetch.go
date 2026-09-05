// Package safefetch 提供面向"不可信 URL"的安全外呼 HTTP 客户端。
//
// 适用场景：URL 来自外部数据（如上游 AI 响应体里的资源链接），且抓取结果
// 最终会透出给终端用户。默认客户端强制：
//   - 仅允许 http/https；host 不得为 localhost/私网/保留地址字面量；
//   - DNS 解析后校验全部解析结果再拨号（防 DNS 重绑定）；
//   - 重定向逐跳重新校验目标地址；
//   - 不继承 HTTP_PROXY 等环境代理，避免经代理绕过内网拦截。
//
// 防护语义与 handler 层订阅导入（safeSubscriptionDialContext）保持一致；
// 管理员显式配置的可信外呼（上游 base_url、告警 webhook 等）不在此约束范围。
package safefetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout        = 30 * time.Second
	dialTimeout           = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 60 * time.Second
	idleConnTimeout       = 90 * time.Second
	maxRedirects          = 5
)

// ErrBlockedHost 表示目标地址解析到被禁止的网络范围（本机/私网/保留段）。
var ErrBlockedHost = errors.New("target host resolves to a blocked address")

// blockedPrefixes 补充 net.IP 私有判断未覆盖的保留/特殊网段
// （CGNAT、文档段、 benchmarks、NAT64、IANA 特殊用途等）。
var blockedPrefixes = []netip.Prefix{
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

// BlockedIP 判断 IP 是否属于禁止访问的范围：未指定/环回/私网/链路本地/
// 组播/4in6 映射地址，以及上方保留网段。解析失败一律视为禁止。
func BlockedIP(ip net.IP) bool {
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
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

// ValidatePublicHTTPURL 校验 URL：格式合法、仅 http/https、host 不为
// localhost/私网/保留地址字面量。解析结果由 DialContext 在拨号时兜底校验。
func ValidatePublicHTTPURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return errors.New("url host is required")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return ErrBlockedHost
	}
	if ip := net.ParseIP(host); ip != nil && BlockedIP(ip) {
		return ErrBlockedHost
	}
	return nil
}

// ValidatePublicHTTPURLResolved 校验 URL，并解析域名后检查全部结果。
// 它适用于无法直接使用 NewClient 的场景，例如必须保留上游账号客户端
// 的代理、TLS 指纹或认证配置时；调用方仍应在请求客户端上配置逐跳校验。
func ValidatePublicHTTPURLResolved(ctx context.Context, raw string) error {
	if err := ValidatePublicHTTPURL(raw); err != nil {
		return err
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("invalid url")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if net.ParseIP(host) != nil {
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return errors.New("failed to resolve target host")
	}
	for _, ip := range ips {
		if BlockedIP(ip) {
			return ErrBlockedHost
		}
	}
	return nil
}

// DialContext 解析 host 并校验全部解析结果，随后直连已校验的 IP
// （而非域名），使"校验"与"拨号"使用同一地址，防 DNS 重绑定。
func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid target host")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("failed to resolve target host")
	}
	for _, ip := range ips {
		if BlockedIP(ip) {
			return nil, ErrBlockedHost
		}
	}
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

// NewClient 构造访问"不可信 URL"的安全客户端。timeout <= 0 时使用默认值。
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           DialContext,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			IdleConnTimeout:       idleConnTimeout,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if err := ValidatePublicHTTPURL(req.URL.String()); err != nil {
				return errors.New("redirect target is not allowed")
			}
			return nil
		},
	}
}
