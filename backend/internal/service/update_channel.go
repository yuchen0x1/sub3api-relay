package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

// UpdateChannel 自更新通道。项目自身发布只使用 Gitee。
type UpdateChannel string

const (
	// UpdateChannelGitHub 保留用于其他仅支持 GitHub 的上游同步任务。
	UpdateChannelGitHub UpdateChannel = "github"
	// UpdateChannelGitee 项目自身发布的唯一通道。
	UpdateChannelGitee UpdateChannel = "gitee"
)

const (
	// GitHubUpdateRepo GitHub 上的自更新仓库
	GitHubUpdateRepo = "yuchen0x1/sub3api-relay"
	// GiteeUpdateRepo Gitee 上的自更新仓库。
	// 注意：Gitee 仓库名与 GitHub 仓库名（yuchen0x1/sub3api-relay）不同，
	// 两者均为官方发布仓库，发布内容保持一致。
	GiteeUpdateRepo = "yuchen0x1/personal-ai-api-gateway-sub3api"
)

const (
	// updateChannelEnvOverride 可选环境变量：强制指定更新通道（github/gitee），
	// 留空或非法值时自动探测。不影响任何现有配置项。
	updateChannelEnvOverride = "UPDATE_CHANNEL"

	// updateChannelGeoTimeout geo 探测超时
	updateChannelGeoTimeout = 3 * time.Second
	// updateChannelRaceTimeout release API 竞速总超时
	updateChannelRaceTimeout = 8 * time.Second
	// updateChannelCacheTTL 通道解析结果的进程内缓存 TTL（与 update_service 的 TTL 缓存风格一致）
	updateChannelCacheTTL = 10 * time.Minute
	// updateChannelMaxBodyBytes 探测响应体上限，防止异常大响应
	updateChannelMaxBodyBytes = 64 * 1024

	// qifuGeoURL 百度千帆 IP 归属地 API（响应 data.country 含 "中国"）
	qifuGeoURL = "https://qifu-api.baidubce.com/ip/local/geo/v1/district"
	// ipSBGeoURL ip.sb geoip API（country_code === "CN"）
	ipSBGeoURL = "https://api.ip.sb/geoip"

	giteeLatestReleaseAPI  = "https://gitee.com/api/v5/repos/" + GiteeUpdateRepo + "/releases/latest"
	githubLatestReleaseAPI = "https://api.github.com/repos/" + GitHubUpdateRepo + "/releases/latest"
)

// updateChannelAllowedHosts 通道探测允许访问的域名白名单（防 SSRF）
var updateChannelAllowedHosts = map[string]struct{}{
	"qifu-api.baidubce.com": {},
	"api.ip.sb":             {},
	"gitee.com":             {},
	"api.github.com":        {},
}

// reservedChannelCIDRs 环回/私有/链路本地之外的保留地址段（同样必须拒绝）
var reservedChannelCIDRs = mustParseChannelCIDRs(
	"0.0.0.0/8",       // 本网络
	"100.64.0.0/10",   // CGNAT 共享地址
	"192.0.0.0/24",    // IETF 协议分配
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // 网络基准测试
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"240.0.0.0/4",     // 保留（含广播 255.255.255.255）
	"2001:db8::/32",   // IPv6 文档段
	"fec0::/10",       // 已废弃的 IPv6 站点本地
)

func mustParseChannelCIDRs(cidrs ...string) []*net.IPNet {
	parsed := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("update_channel: invalid reserved CIDR %q: %v", cidr, err))
		}
		parsed = append(parsed, ipNet)
	}
	return parsed
}

// isForbiddenChannelIP 判断 IP 是否为 localhost/环回/私有/链路本地/保留地址
func isForbiddenChannelIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, cidr := range reservedChannelCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// validateChannelURL 安全约束：仅 http/https + 域名白名单，且不允许 userinfo
func validateChannelURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid update channel URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("update channel URL scheme not allowed: %s", parsed.Scheme)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("update channel URL must not contain userinfo")
	}
	host := strings.ToLower(parsed.Hostname())
	if _, ok := updateChannelAllowedHosts[host]; !ok {
		return nil, fmt.Errorf("update channel host not allowed: %s", host)
	}
	return parsed, nil
}

// resolveChannelHost 先解析目标域名，再校验全部解析结果均为公网地址（防 SSRF/内网探测）
func resolveChannelHost(host string) error {
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("update channel host lookup failed for %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("update channel host %s resolved to no address", host)
	}
	for _, ipStr := range ips {
		if isForbiddenChannelIP(net.ParseIP(ipStr)) {
			return fmt.Errorf("update channel host %s resolved to forbidden address %s", host, ipStr)
		}
	}
	return nil
}

// updateChannelDialer 连接建立时再次校验目标 IP（防 DNS rebinding）。
// 双重校验：LookupHost 预解析 + 拨号时 Control 回调。
var updateChannelDialer = &net.Dialer{
	Timeout:   updateChannelGeoTimeout,
	KeepAlive: 30 * time.Second,
	Control: func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("update channel: invalid dial address %q: %w", address, err)
		}
		if isForbiddenChannelIP(net.ParseIP(host)) {
			return fmt.Errorf("update channel: dial to forbidden address blocked: %s", host)
		}
		return nil
	},
}

// updateChannelHTTPClient 通道探测专用客户端。
// 注意：刻意不使用代理 —— geo 探测必须反映本机真实网络位置，
// 走代理会把代理出口的归属地误判为本机归属地。
var updateChannelHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 nil,
		DialContext:           updateChannelDialer.DialContext,
		TLSHandshakeTimeout:   updateChannelGeoTimeout,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// channelSafeGet 白名单 + IP 校验后发起 GET（限制超时与响应体大小）
func channelSafeGet(ctx context.Context, rawURL string, timeout time.Duration) ([]byte, error) {
	parsed, err := validateChannelURL(rawURL)
	if err != nil {
		return nil, err
	}
	if err := resolveChannelHost(parsed.Hostname()); err != nil {
		return nil, err
	}

	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(getCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Sub3API-Updater")
	req.Header.Set("Accept", "application/json")

	resp, err := updateChannelHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("probe %s returned HTTP %d", parsed.Host, resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, updateChannelMaxBodyBytes))
}

// 通道解析结果进程内缓存（带 TTL 的包级变量）
var (
	updateChannelMu    sync.Mutex
	updateChannelCache UpdateChannel
	updateChannelExpAt time.Time
)

// ResolveUpdateChannel 解析当前应使用的更新通道。
//
// 项目自身的版本发布只在 Gitee 维护。这里不再根据 IP、DNS 或外部
// 可达性在 Gitee/GitHub 之间切换，避免 GitHub 不可达或没有 Release 时
// 把真实的 Gitee 新版本误报为“已是最新版本”。
func ResolveUpdateChannel(ctx context.Context) (UpdateChannel, error) {
	_ = ctx
	return UpdateChannelGitee, nil
}

// channelFromEnv 读取 UPDATE_CHANNEL 环境变量强制指定的通道
func channelFromEnv() UpdateChannel {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(updateChannelEnvOverride))) {
	case string(UpdateChannelGitHub):
		return UpdateChannelGitHub
	case string(UpdateChannelGitee):
		return UpdateChannelGitee
	}
	return ""
}

// detectUpdateChannel 实际执行通道探测
func detectUpdateChannel(ctx context.Context) UpdateChannel {
	// 1) 百度千帆 geo
	if body, err := channelSafeGet(ctx, qifuGeoURL, updateChannelGeoTimeout); err == nil {
		return channelFromChinaText(string(body))
	}
	// 2) ip.sb geoip
	if body, err := channelSafeGet(ctx, ipSBGeoURL, updateChannelGeoTimeout); err == nil {
		return channelFromIPSBText(body)
	}
	// 3) 竞速 Gitee / GitHub release API，先成功者胜
	if ch, ok := raceUpdateChannelReachability(ctx); ok {
		return ch
	}
	// 4) 全部失败：github 兜底
	return UpdateChannelGitHub
}

// channelFromChinaText qifu 响应包含 "中国"/"China" → gitee，否则 github
func channelFromChinaText(body string) UpdateChannel {
	if strings.Contains(body, "中国") || strings.Contains(strings.ToLower(body), "china") {
		return UpdateChannelGitee
	}
	return UpdateChannelGitHub
}

// channelFromIPSBText ip.sb 响应 country_code === "CN" → gitee，否则 github
func channelFromIPSBText(body []byte) UpdateChannel {
	var geo struct {
		CountryCode string `json:"country_code"`
	}
	if err := json.Unmarshal(body, &geo); err == nil && strings.EqualFold(geo.CountryCode, "CN") {
		return UpdateChannelGitee
	}
	// 容错：JSON 解析失败时退回文本匹配
	if strings.Contains(string(body), `"country_code":"CN"`) {
		return UpdateChannelGitee
	}
	return UpdateChannelGitHub
}

// raceUpdateChannelReachability 并发探测 Gitee 与 GitHub 的 releases/latest，先成功者胜
func raceUpdateChannelReachability(ctx context.Context) (UpdateChannel, bool) {
	rctx, cancel := context.WithTimeout(ctx, updateChannelRaceTimeout)
	defer cancel()

	results := make(chan UpdateChannel, 2)
	go func() {
		if _, err := channelSafeGet(rctx, giteeLatestReleaseAPI, updateChannelRaceTimeout); err == nil {
			results <- UpdateChannelGitee
		}
	}()
	go func() {
		if _, err := channelSafeGet(rctx, githubLatestReleaseAPI, updateChannelRaceTimeout); err == nil {
			results <- UpdateChannelGitHub
		}
	}()

	select {
	case ch := <-results:
		return ch, true
	case <-rctx.Done():
		return "", false
	}
}
