//go:build unit

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

type antigravityBodyLogSink struct{ messages []string }

func (s *antigravityBodyLogSink) WriteLogEvent(event *logger.LogEvent) {
	s.messages = append(s.messages, event.Message)
}

func TestAntigravityResponseBodyLogsOmitUpstreamSecrets(t *testing.T) {
	logger.InitBootstrap()
	for _, mode := range []string{"single_account_exhausted", "long_delay", "mapped_error"} {
		t.Run(mode, func(t *testing.T) {
			var standardLogs bytes.Buffer
			previousWriter := log.Writer()
			log.SetOutput(&standardLogs)
			t.Cleanup(func() { log.SetOutput(previousWriter) })
			sink := &antigravityBodyLogSink{}
			logger.SetSink(sink)
			t.Cleanup(func() { logger.SetSink(nil) })
			delay := "0.1s"
			if mode == "long_delay" {
				delay = "10s"
			}
			body := []byte(fmt.Sprintf(`{"error":{"message":"echoed-secret-bearer-token","code":503,"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"gemini-3-flash"},"reason":"RATE_LIMIT_EXCEEDED"},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":%q}]}}`, delay))
			account := &Account{ID: 42, Type: AccountTypeOAuth, Platform: PlatformAntigravity}
			if mode == "mapped_error" {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				svc := &AntigravityGatewayService{settingService: &SettingService{cfg: &config.Config{Gateway: config.GatewayConfig{LogUpstreamErrorBody: true, LogUpstreamErrorBodyMaxBytes: 4096}}}}
				require.Error(t, svc.writeMappedClaudeError(c, account, http.StatusServiceUnavailable, "request-1", body))
			} else {
				ctx := context.Background()
				if mode == "single_account_exhausted" {
					ctx = ctxWithSingleAccountRetry()
				}
				response := func() *http.Response {
					return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body))}
				}
				params := antigravityRetryLoopParams{ctx: ctx, prefix: "[body-log-test]", account: account, accessToken: "request-secret", action: "generateContent", body: []byte(`{}`), accountRepo: &stubAntigravityAccountRepo{}, httpUpstream: &mockSmartRetryUpstream{responses: []*http.Response{response()}, errors: []error{nil}, repeatLast: true}, handleError: func(context.Context, string, *Account, int, http.Header, []byte, string, int64, string, bool) *handleModelRateLimitResult {
					return nil
				}}
				result := (&AntigravityGatewayService{}).handleSmartRetry(params, response(), body, "https://ag.test", 0, []string{"https://ag.test"})
				require.NotNil(t, result)
				if mode == "single_account_exhausted" {
					require.NotNil(t, result.resp)
					returned, err := io.ReadAll(result.resp.Body)
					require.NoError(t, err)
					require.Equal(t, body, returned, "log hardening must preserve the upstream response")
				} else {
					require.NotNil(t, result.switchError)
				}
			}
			logs := standardLogs.String() + strings.Join(sink.messages, "\n")
			require.NotContains(t, logs, "echoed-secret-bearer-token")
			require.NotContains(t, logs, "request-secret")
			require.Contains(t, logs, "status=503")
			require.Contains(t, logs, fmt.Sprintf("body_bytes=%d", len(body)))
		})
	}
}
