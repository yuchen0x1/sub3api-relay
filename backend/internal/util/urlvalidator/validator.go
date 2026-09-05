package urlvalidator

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ValidateURLFormat 校验出站 URL 的基本格式：可解析、scheme 为 http/https、host 非空、端口合法。
// 不做域名白名单、内网地址或 DNS 解析层面的限制，上游地址完全由管理员自行决定。
func ValidateURLFormat(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid url: %s", trimmed)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", fmt.Errorf("invalid url scheme: %s", parsed.Scheme)
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", errors.New("invalid host")
	}

	if port := parsed.Port(); port != "" {
		num, err := strconv.Atoi(port)
		if err != nil || num <= 0 || num > 65535 {
			return "", fmt.Errorf("invalid port: %s", port)
		}
	}

	return strings.TrimRight(trimmed, "/"), nil
}
