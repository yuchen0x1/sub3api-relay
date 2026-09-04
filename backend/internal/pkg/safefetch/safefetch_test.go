package safefetch

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidatePublicHTTPURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"public https", "https://example.com/a.png", false},
		{"public http", "http://example.com/a.png", false},
		{"public ip", "http://8.8.8.8/a.png", false},
		{"empty scheme", "example.com/a.png", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/a", true},
		{"no host", "http:///a.png", true},
		{"localhost", "http://localhost:8080/x", true},
		{"subdomain localhost", "http://api.localhost/x", true},
		{"loopback v4", "http://127.0.0.1/x", true},
		{"loopback v6", "http://[::1]/x", true},
		{"private v4", "http://10.1.2.3/x", true},
		{"private v4 192", "http://192.168.1.1/x", true},
		{"link local", "http://169.254.169.254/latest/meta-data/", true},
		{"cgnat", "http://100.64.0.1/x", true},
		{"unspecified", "http://0.0.0.0/x", true},
		{"mapped loopback", "http://[::ffff:127.0.0.1]/x", true},
		{"ula v6", "http://[fd00::1]/x", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePublicHTTPURL(tc.raw)
			if tc.wantErr != (err != nil) {
				t.Fatalf("ValidatePublicHTTPURL(%q) = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
		})
	}
}

func TestBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.1.1",
		"0.0.0.0", "100.64.0.1", "198.18.0.1", "224.0.0.1", "::1", "fe80::1",
		"::ffff:127.0.0.1", "fd00::1", "2001:db8::1",
	}
	for _, s := range blocked {
		if !BlockedIP(net.ParseIP(s)) {
			t.Errorf("BlockedIP(%s) = false, want true", s)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"}
	for _, s := range allowed {
		if BlockedIP(net.ParseIP(s)) {
			t.Errorf("BlockedIP(%s) = true, want false", s)
		}
	}
}

// 环回地址上的 httptest 服务本身就在被拦截范围内，因此直接请求即应被
// DialContext 拒绝；再把请求 302 到本机另一个地址，验证重定向同样被拦截。
func TestClientBlocksLoopbackAndRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/redirected", http.StatusFound)
	}))
	defer srv.Close()

	client := NewClient(0)
	resp, err := client.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("request to loopback target should be blocked by DialContext")
	}
	t.Logf("dial guard error: %v", err)

	// ValidatePublicHTTPURL 在拨号前就应拦截环回目标（含重定向目标）。
	if err := ValidatePublicHTTPURL("http://127.0.0.1:1/redirected"); err == nil {
		t.Fatal("redirect target 127.0.0.1 should fail validation")
	}
}

// CheckRedirect 的拦截逻辑单独验证：5 跳上限 + 目标校验。
func TestCheckRedirectPolicy(t *testing.T) {
	client := NewClient(0)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err := client.CheckRedirect(req, make([]*http.Request, maxRedirects)); err == nil {
		t.Fatal("expected too-many-redirects error")
	}
	req2, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/x", nil)
	if err := client.CheckRedirect(req2, nil); err == nil {
		t.Fatal("expected blocked redirect target error")
	}
}
