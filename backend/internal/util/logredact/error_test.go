package logredact

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestErrorSummary_DoesNotLogCredentialsOrUpstreamBody(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want []string
	}{
		{"proxy_parse", &url.Error{Op: "parse", URL: "http://proxy-user:proxy-password@bad%host", Err: errors.New("invalid URL escape")}, []string{"kind=invalid_url", "type=*url.Error"}},
		{"request_timeout", &url.Error{Op: "Get", URL: "https://upstream.example/api?key=secret-api-key", Err: context.DeadlineExceeded}, []string{"kind=deadline_exceeded"}},
		{"dns", fmt.Errorf("request failed: %w", &net.DNSError{Name: "sensitive-host", Err: "no such host"}), []string{"kind=dns_error"}},
		{"oauth_body", errors.New(`token refresh failed (HTTP 400): {"error":"invalid_grant","refresh_token":"oauth-secret","unexpected":"unlabelled-secret"}`), []string{"http_status=400", "code=invalid_grant"}},
		{"provider_body", errors.New(`HTTP status 403: unexpected echo unlabelled-secret Authorization: Bearer bearer-secret`), []string{"http_status=403"}},
		{"canceled", fmt.Errorf("credential secret-api-key: %w", context.Canceled), []string{"kind=canceled"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := ErrorSummary(tc.err)
			for _, secret := range []string{"proxy-user", "proxy-password", "secret-api-key", "sensitive-host", "oauth-secret", "unlabelled-secret", "bearer-secret"} {
				if strings.Contains(out, secret) {
					t.Fatalf("summary leaked %q: %s", secret, out)
				}
			}
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("summary %q lacks diagnostic %q", out, want)
				}
			}
		})
	}
	if got := ErrorSummary(nil); got != "" {
		t.Fatalf("nil error summary = %q", got)
	}
}
