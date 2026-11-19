package logredact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var diagnosticHTTPStatus = regexp.MustCompile(`(?i)\b(?:http(?:\s+status)?|status(?:\s+code)?)[\s:=()]+([1-5][0-9]{2})\b`)

// ErrorSummary describes a failure without copying its free-form message. Network
// errors can contain complete URLs (including proxy userinfo and API keys), and
// upstream error bodies can echo credentials even without a recognizable label.
// The caller's event identifies the operation; the type, class, HTTP status and
// known provider code below retain diagnostics without logging those payloads.
// This is for logs only: callers keep the original error for control flow.
func ErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	kind := "upstream_error"
	var netErr net.Error
	var dnsErr *net.DNSError
	var urlErr *url.Error
	switch {
	case errors.Is(err, context.Canceled):
		kind = "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		kind = "deadline_exceeded"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		kind = "unexpected_eof"
	case errors.As(err, &dnsErr):
		kind = "dns_error"
	case errors.As(err, &netErr) && netErr.Timeout():
		kind = "timeout"
	case errors.As(err, &urlErr) && urlErr.Op == "parse":
		kind = "invalid_url"
	case errors.As(err, &netErr):
		kind = "network_error"
	}
	message := strings.ToLower(err.Error())
	if kind == "upstream_error" || kind == "network_error" {
		switch {
		case strings.Contains(message, "invalid proxy url"), strings.Contains(message, "proxy url missing host"), strings.Contains(message, "unsupported proxy"), strings.Contains(message, "unsupported vless"):
			kind = "proxy_configuration_error"
		case strings.Contains(message, "connection refused"):
			kind = "connection_refused"
		case strings.Contains(message, "tls handshake"), strings.Contains(message, "certificate"), strings.Contains(message, "x509:"):
			kind = "tls_error"
		}
	}
	result := fmt.Sprintf("type=%T kind=%s", err, kind)
	// Only fixed vocabulary and three-digit status codes leave this function.
	// Never return a substring of an unknown provider message or request URL.
	if status := diagnosticHTTPStatus.FindStringSubmatch(message); len(status) == 2 {
		result += " http_status=" + status[1]
	}
	for _, code := range []string{
		"invalid_grant", "invalid_client", "unauthorized_client", "access_denied",
		"invalid_api_key", "insufficient_quota", "rate_limit_exceeded",
		"permission_denied", "unauthenticated", "resource_exhausted",
	} {
		if strings.Contains(message, code) {
			result += " code=" + code
			break
		}
	}
	return result
}
