package proxyurl

import "testing"

func TestParseRejectsHysteria2(t *testing.T) {
	_, _, err := Parse("hysteria2://token@example.com:443?insecure=1")
	if err == nil {
		t.Fatal("hysteria2 must be rejected")
	}
}
