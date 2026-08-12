package proxyurl

import "testing"

func TestParseHysteria2(t *testing.T) {
	trimmed, parsed, err := Parse("hysteria2://token@example.com:443?insecure=1")
	if err != nil {
		t.Fatalf("hysteria2 should parse: %v", err)
	}
	if trimmed == "" || parsed.Scheme != "hysteria2" {
		t.Fatalf("unexpected parse result: %q %#v", trimmed, parsed)
	}
}
