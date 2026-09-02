package repository

import "strings"

// isTunnelProxyScheme reports whether the proxy scheme needs a custom dialer
// instead of the http/socks5 proxy support built into HTTP client libraries.
func isTunnelProxyScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "vless", "ss", "trojan", "vmess":
		return true
	}
	return false
}
