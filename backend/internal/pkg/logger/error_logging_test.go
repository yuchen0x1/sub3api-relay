package logger

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLegacyPrintf_SummarizesSensitiveErrors(t *testing.T) {
	failure := errors.New(`refresh failed (HTTP 400): invalid_grant secret-proxy-password unexpected-raw-token`)
	for _, initialized := range []bool{false, true} {
		name := "bootstrap"
		if initialized {
			name = "structured"
		}
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			previousWriter := log.Writer()
			previous := global.Load()
			defer func() {
				global.Store(previous)
				log.SetOutput(previousWriter)
			}()
			log.SetOutput(&output)
			core, observed := observer.New(zap.DebugLevel)
			if initialized {
				global.Store(zap.New(core))
			} else {
				global.Store(nil)
			}
			LegacyPrintf("test", "refresh failed account_id=%d api_key_id=%d api_key_enabled=%t error=%v", 42, int64(7), true, failure)
			got := output.String()
			if initialized {
				entries := observed.All()
				if len(entries) != 1 {
					t.Fatalf("got %d entries", len(entries))
				}
				got = entries[0].Message
			}
			for _, secret := range []string{"secret-proxy-password", "unexpected-raw-token"} {
				if strings.Contains(got, secret) {
					t.Fatalf("log leaked %q: %s", secret, got)
				}
			}
			for _, want := range []string{"account_id=42", "api_key_id=7", "api_key_enabled=true", "http_status=400", "code=invalid_grant"} {
				if !strings.Contains(got, want) {
					t.Fatalf("log lacks %q: %s", want, got)
				}
			}
		})
	}
}
