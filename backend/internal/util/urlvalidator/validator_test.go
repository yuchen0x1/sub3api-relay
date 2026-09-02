package urlvalidator

import "testing"

func TestValidateURLFormat(t *testing.T) {
	if _, err := ValidateURLFormat(""); err == nil {
		t.Fatalf("expected empty url to fail")
	}
	if _, err := ValidateURLFormat("://bad"); err == nil {
		t.Fatalf("expected invalid url to fail")
	}
	if _, err := ValidateURLFormat("ftp://example.com"); err == nil {
		t.Fatalf("expected non-http scheme to fail")
	}
	if _, err := ValidateURLFormat("https://example.com"); err != nil {
		t.Fatalf("expected https to pass, got %v", err)
	}
	if _, err := ValidateURLFormat("https://example.com:bad"); err == nil {
		t.Fatalf("expected invalid port to fail")
	}

	// 验证末尾斜杠被移除
	normalized, err := ValidateURLFormat("https://example.com/")
	if err != nil {
		t.Fatalf("expected trailing slash url to pass, got %v", err)
	}
	if normalized != "https://example.com" {
		t.Fatalf("expected trailing slash to be removed, got %s", normalized)
	}

	// 验证多个末尾斜杠被移除
	normalized, err = ValidateURLFormat("https://example.com///")
	if err != nil {
		t.Fatalf("expected multiple trailing slashes to pass, got %v", err)
	}
	if normalized != "https://example.com" {
		t.Fatalf("expected all trailing slashes to be removed, got %s", normalized)
	}

	// 验证带路径的 URL 末尾斜杠被移除
	normalized, err = ValidateURLFormat("https://example.com/api/v1/")
	if err != nil {
		t.Fatalf("expected trailing slash url with path to pass, got %v", err)
	}
	if normalized != "https://example.com/api/v1" {
		t.Fatalf("expected trailing slash to be removed from path, got %s", normalized)
	}
}

// 上游地址不再受域名白名单、内网拦截或 http 明文限制：
// 任意公网域名、第三方中转、localhost/局域网自建网关、http 明文均可用。
func TestValidateURLFormatAllowsAnyUpstream(t *testing.T) {
	cases := []string{
		"https://token.sensenova.cn/v1", // 任意第三方公网域名
		"https://relay.example.test/v1", // 未收录的自定义域名
		"http://example.com/v1",         // http 明文
		"http://127.0.0.1:8080/v1",      // 回环地址
		"http://localhost:3000",         // localhost
		"http://192.168.1.10:3000/v1",   // 局域网 IPv4
		"https://10.0.0.5:8443",         // 私网 IPv4
		"https://[fd00::1]:9000/v1",     // IPv6 ULA
	}
	for _, raw := range cases {
		if _, err := ValidateURLFormat(raw); err != nil {
			t.Fatalf("expected %q to pass, got %v", raw, err)
		}
	}
}
