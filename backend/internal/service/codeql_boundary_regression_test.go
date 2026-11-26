package service

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func TestQuotaResetPreconversionBounds(t *testing.T) {
	for _, value := range []any{int64(math.MaxInt64), json.Number("4294967297"), "NaN", math.Inf(1), "-24"} {
		extra := map[string]any{
			"quota_daily_reset_mode": "fixed", "quota_weekly_reset_mode": "fixed",
			"quota_daily_limit": 1, "quota_weekly_limit": 1,
			"quota_daily_reset_hour": value, "quota_weekly_reset_hour": value, "quota_weekly_reset_day": value,
		}
		require.Error(t, ValidateQuotaResetConfig(extra))
		ComputeQuotaResetAt(extra)
		NormalizeFixedQuotaWindows(extra)
		for _, key := range []string{"quota_daily_reset_at", "quota_weekly_reset_at", "quota_daily_start", "quota_weekly_start"} {
			raw, ok := extra[key].(string)
			require.True(t, ok, key)
			stamp, err := time.Parse(time.RFC3339, raw)
			require.NoError(t, err, key)
			require.Zero(t, stamp.Hour(), key)
			if strings.Contains(key, "weekly") {
				require.Equal(t, time.Monday, stamp.Weekday(), key)
			}
		}
	}
	value, ok := quotaResetComponent(json.Number("23.5"), 23)
	require.True(t, ok)
	require.Equal(t, 23, value)
}

func TestOpenAIWSReplayInputBudget(t *testing.T) {
	// Reuse a backing item to model accumulated history without allocating the full budget.
	prior := make([]json.RawMessage, 64)
	item := make(json.RawMessage, 1<<20)
	for i := range prior {
		prior[i] = item
	}
	payload := []byte(`{"input":[{"type":"input_text","text":"current"}]}`)
	history, exists, err := buildOpenAIWSReplayInputSequence(prior, true, payload, true)
	require.ErrorIs(t, err, errOpenAIWSReplayInputTooLarge)
	require.False(t, exists)
	require.Nil(t, history)

	// An independent turn can seed a fresh history after a previous chain overflows.
	history, exists, err = buildOpenAIWSReplayInputSequence(prior, true, payload, false)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, history, 1)

	// Output from the upstream turn shares the same budget as incoming messages.
	_, err = cloneBoundedOpenAIWSReplayInput(history, prior)
	require.ErrorIs(t, err, errOpenAIWSReplayInputTooLarge)
}

func TestOpenAIResponseProtocolsUseSafeMediaType(t *testing.T) {
	body := `{"id":"response_1","data":[]}`
	for _, tc := range []struct {
		name  string
		write func(*OpenAIGatewayService, *http.Response, *gin.Context) error
	}{
		{"images", func(s *OpenAIGatewayService, resp *http.Response, c *gin.Context) error {
			_, _, _, err := s.handleOpenAIImagesNonStreamingResponse(resp, c)
			return err
		}},
		{"compact passthrough", func(s *OpenAIGatewayService, resp *http.Response, c *gin.Context) error {
			_, err := s.handleNonStreamingResponsePassthrough(context.Background(), resp, c, "", "")
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, tt := range []struct{ upstream, expected string }{
				{"text/html; charset=utf-8", "application/json"},
				{"image/svg+xml", "application/json"},
				{"application/xhtml+xml", "application/json"},
				{"invalid content type", "application/json"},
				{"application/vnd.api+json; charset=utf-8", "application/vnd.api+json; charset=utf-8"},
			} {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
				resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {tt.upstream}}, Body: io.NopCloser(strings.NewReader(body))}
				require.NoError(t, tc.write(&OpenAIGatewayService{cfg: &config.Config{}}, resp, c))
				require.Equal(t, tt.expected, recorder.Header().Get("Content-Type"))
				require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
				require.Equal(t, body, recorder.Body.String())

				recorder = httptest.NewRecorder()
				c, _ = gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
				activeBody := `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(document.cookie)"></svg>`
				resp = &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {tt.upstream}}, Body: io.NopCloser(strings.NewReader(activeBody))}
				require.NoError(t, tc.write(&OpenAIGatewayService{cfg: &config.Config{}}, resp, c))
				require.Equal(t, tt.expected, recorder.Result().Header.Get("Content-Type"))
				require.Equal(t, "nosniff", recorder.Result().Header.Get("X-Content-Type-Options"))
				require.Equal(t, activeBody, recorder.Body.String(), "API payloads must not be HTML-escaped or rewritten")
			}
		})
	}
}

func TestGatewayDiagnosticHeadersRedactAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "https://api.example.test/v1/messages", nil)
	request.Header.Set("Authorization", "Bearer secret-bearer-value")
	request.Header.Set("X-Api-Key", "secret-api-key-value")
	request.Header.Set("User-Agent", "regression-client")
	line := buildClaudeMimicDebugLine(request, nil, &Account{ID: 42}, "oauth", false)
	require.NotContains(t, line, "secret-bearer-value")
	require.NotContains(t, line, "secret-api-key-value")
	require.Contains(t, line, `authorization="Bearer [redacted]"`)
	require.Contains(t, line, `x-api-key="[redacted]"`)
	require.Contains(t, line, `user-agent="regression-client"`)
}
