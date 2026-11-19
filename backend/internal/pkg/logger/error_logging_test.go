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
			LegacyPrintf("test", "refresh failed account_id=%d error=%v", 42, failure)
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
			for _, want := range []string{"account_id=42", "http_status=400", "code=invalid_grant"} {
				if !strings.Contains(got, want) {
					t.Fatalf("log lacks %q: %s", want, got)
				}
			}
		})
	}
}
