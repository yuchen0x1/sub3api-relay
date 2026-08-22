//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBarkAlertProvider_Send(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotContentType string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewBarkAlertProvider()
	p.SetConfig(OpsBarkNotificationConfig{
		ServerURL: server.URL,
		Key:       "test-key-123",
	})

	msg := AlertNotificationMessage{
		Title:    "[Ops Alert][critical] balance low",
		Body:     "line1\nline2",
		Severity: "critical",
	}
	err := p.Send(context.Background(), msg)
	require.NoError(t, err)

	require.Equal(t, "/test-key-123", gotPath)
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, "[Ops Alert][critical] balance low", gotPayload["title"])
	require.Equal(t, "line1\nline2", gotPayload["body"])
	require.Equal(t, "ops-alert", gotPayload["group"])
}

func TestBarkAlertProvider_MissingKey(t *testing.T) {
	t.Parallel()

	p := NewBarkAlertProvider()
	p.SetConfig(OpsBarkNotificationConfig{ServerURL: "https://api.day.app"})
	err := p.Send(context.Background(), AlertNotificationMessage{Title: "x"})
	require.Error(t, err)
}

func TestBarkAlertProvider_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewBarkAlertProvider()
	p.SetConfig(OpsBarkNotificationConfig{ServerURL: server.URL, Key: "k"})
	err := p.Send(context.Background(), AlertNotificationMessage{Title: "x"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestWebhookAlertProvider_Send(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotAuth string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("X-Auth-Token")
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewWebhookAlertProvider()
	p.SetConfig(OpsWebhookNotificationConfig{
		URL:     server.URL + "/dingtalk",
		Headers: map[string]string{"X-Auth-Token": "secret-abc"},
	})

	msg := AlertNotificationMessage{
		Title:       "[Ops Alert][warning] group degraded",
		Body:        "body-text",
		Severity:    "warning",
		RuleName:    "group_available_ratio",
		MetricType:  "group_available_ratio",
		Operator:    "<",
		Status:      "firing",
		EventID:     42,
		HasMetric:   true,
		MetricValue: 12.5,
		Threshold:   20,
		FiredAt:     time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC),
	}
	err := p.Send(context.Background(), msg)
	require.NoError(t, err)

	require.Equal(t, "/dingtalk", gotPath)
	require.Equal(t, "secret-abc", gotAuth)
	require.Equal(t, "ops_alert", gotPayload["source"])
	require.Equal(t, "warning", gotPayload["severity"])
	require.Equal(t, "group_available_ratio", gotPayload["metric_type"])
	require.Equal(t, float64(12.5), gotPayload["metric_value"])
	require.Equal(t, float64(20), gotPayload["threshold_value"])
	require.Equal(t, "2026-08-22T03:00:00Z", gotPayload["fired_at"])
}

func TestWebhookAlertProvider_MissingURL(t *testing.T) {
	t.Parallel()

	p := NewWebhookAlertProvider()
	p.SetConfig(OpsWebhookNotificationConfig{})
	err := p.Send(context.Background(), AlertNotificationMessage{Title: "x"})
	require.Error(t, err)
}

func TestBuildAlertNotificationMessage(t *testing.T) {
	t.Parallel()

	mv := 3.5
	tv := 5.0
	rule := &OpsAlertRule{
		Name:       "balance low",
		Severity:   "warning",
		MetricType: "group_available_accounts",
		Operator:   "<",
		Threshold:  5,
	}
	event := &OpsAlertEvent{
		ID:             9,
		Severity:       "warning",
		Status:         "firing",
		MetricValue:    &mv,
		ThresholdValue: &tv,
		FiredAt:        time.Date(2026, 8, 22, 3, 0, 0, 0, time.UTC),
		Description:    "channel x disabled",
	}

	msg := buildAlertNotificationMessage(rule, event)
	require.Equal(t, "[Ops Alert][warning] balance low", msg.Title)
	require.Equal(t, "warning", msg.Severity)
	require.Equal(t, float64(3.5), msg.MetricValue)
	require.Equal(t, float64(5), msg.Threshold)
	require.True(t, msg.HasMetric)
	require.Contains(t, msg.Body, "channel x disabled")
	require.Contains(t, msg.Body, "threshold 5.00")
	require.Contains(t, msg.Body, "2026-08-22T03:00:00Z")
}

func TestShouldSendOpsAlertByMinSeverity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		minSeverity string
		ruleSev     string
		want        bool
	}{
		{"empty means all", "", "info", true},
		{"empty means all critical", "", "P0", true},
		{"critical min passes critical", "critical", "P0", true},
		{"critical min blocks warning", "critical", "warning", false},
		{"warning min passes warning", "warning", "P1", true},
		{"warning min blocks info", "warning", "info", false},
		{"info min passes info", "info", "info", true},
		{"case insensitive", "CRITICAL", "p0", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldSendOpsAlertByMinSeverity(tc.minSeverity, tc.ruleSev)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestValidateOpsNotificationChannels(t *testing.T) {
	t.Parallel()

	// Bark enabled without key -> error.
	err := validateOpsNotificationChannels(&OpsNotificationChannelsConfig{
		Bark: OpsBarkNotificationConfig{Enabled: true},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bark.key")

	// Webhook enabled with non-http URL -> error.
	err = validateOpsNotificationChannels(&OpsNotificationChannelsConfig{
		Webhook: OpsWebhookNotificationConfig{Enabled: true, URL: "ftp://x"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")

	// Invalid min severity -> error.
	err = validateOpsNotificationChannels(&OpsNotificationChannelsConfig{
		Bark: OpsBarkNotificationConfig{Enabled: true, Key: "k", MinSeverity: "urgent"},
	})
	require.Error(t, err)

	// Valid config passes.
	err = validateOpsNotificationChannels(&OpsNotificationChannelsConfig{
		Bark:    OpsBarkNotificationConfig{Enabled: true, Key: "k", MinSeverity: "warning"},
		Webhook: OpsWebhookNotificationConfig{Enabled: true, URL: "https://example.com/hook"},
	})
	require.NoError(t, err)
}

// fakeOpsNotifierService 提供 GetEmailNotificationConfig 的最小实现。
type fakeOpsNotifierService struct {
	cfg *OpsEmailNotificationConfig
}

func (f *fakeOpsNotifierService) GetEmailNotificationConfig(ctx context.Context) (*OpsEmailNotificationConfig, error) {
	if f == nil || f.cfg == nil {
		return defaultOpsEmailNotificationConfig(), nil
	}
	return f.cfg, nil
}

func (f *fakeOpsNotifierService) GetOpsAlertRuntimeSettings(ctx context.Context) (*OpsAlertRuntimeSettings, error) {
	return defaultOpsAlertRuntimeSettings(), nil
}

func (f *fakeOpsNotifierService) IsMonitoringEnabled(ctx context.Context) bool { return true }

func (f *fakeOpsNotifierService) IsAlertSilenced(ctx context.Context, ruleID int64, platform string, groupID *int64, region *string, now time.Time) (bool, error) {
	return false, nil
}

func (f *fakeOpsNotifierService) GetAccountAvailability(ctx context.Context, platform string, groupID *int64) (*OpsAccountAvailability, error) {
	return nil, nil
}

func TestOpsAlertEvaluator_DispatchFansOutToChannels(t *testing.T) {
	t.Parallel()

	barkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer barkServer.Close()
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = false
	cfg.Channels.Bark = OpsBarkNotificationConfig{
		Enabled:   true,
		ServerURL: barkServer.URL,
		Key:       "bark-key",
	}
	cfg.Channels.Webhook = OpsWebhookNotificationConfig{
		Enabled: true,
		URL:     webhookServer.URL,
	}

	opsSvc := &fakeOpsNotifierService{cfg: cfg}
	evaluator := NewOpsAlertEvaluatorService(nil, nil, nil, nil, nil, nil)
	evaluator.opsService = opsSvc

	rule := &OpsAlertRule{
		ID:          7,
		Name:        "balance low",
		Severity:    "critical",
		MetricType:  "group_available_accounts",
		Operator:    "<",
		Threshold:   1,
		NotifyEmail: false,
	}
	event := &OpsAlertEvent{
		ID:     1,
		Status: "firing",
	}

	sent := evaluator.maybeDispatchAlert(context.Background(), nil, rule, event)
	require.True(t, sent)
}

func TestOpsAlertEvaluator_DispatchNoChannelsReturnsFalse(t *testing.T) {
	t.Parallel()

	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = false
	opsSvc := &fakeOpsNotifierService{cfg: cfg}
	evaluator := NewOpsAlertEvaluatorService(nil, nil, nil, nil, nil, nil)
	evaluator.opsService = opsSvc

	rule := &OpsAlertRule{ID: 1, Name: "r", Severity: "info", NotifyEmail: false}
	event := &OpsAlertEvent{ID: 1, Status: "firing"}
	require.False(t, evaluator.maybeDispatchAlert(context.Background(), nil, rule, event))
}

func TestBarkProvider_SeverityFilterBlocksSend(t *testing.T) {
	t.Parallel()

	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = false
	cfg.Channels.Bark = OpsBarkNotificationConfig{
		Enabled:     true,
		ServerURL:   server.URL,
		Key:         "k",
		MinSeverity: "critical",
	}
	opsSvc := &fakeOpsNotifierService{cfg: cfg}
	evaluator := NewOpsAlertEvaluatorService(nil, nil, nil, nil, nil, nil)
	evaluator.opsService = opsSvc

	rule := &OpsAlertRule{ID: 1, Name: "r", Severity: "info", NotifyEmail: false}
	event := &OpsAlertEvent{ID: 1, Status: "firing"}
	require.False(t, evaluator.maybeDispatchAlert(context.Background(), nil, rule, event))
	require.False(t, hit, "bark should not receive below-min-severity alerts")
}

func TestBarkProvider_RateLimitBlocksSend(t *testing.T) {
	t.Parallel()

	hit := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := defaultOpsEmailNotificationConfig()
	cfg.Alert.Enabled = false
	cfg.Channels.Bark = OpsBarkNotificationConfig{
		Enabled:          true,
		ServerURL:        server.URL,
		Key:              "k",
		RateLimitPerHour: 1,
	}
	opsSvc := &fakeOpsNotifierService{cfg: cfg}
	evaluator := NewOpsAlertEvaluatorService(nil, nil, nil, nil, nil, nil)
	evaluator.opsService = opsSvc

	rule := &OpsAlertRule{ID: 1, Name: "r", Severity: "info", NotifyEmail: false}
	event := &OpsAlertEvent{ID: 1, Status: "firing"}
	require.True(t, evaluator.maybeDispatchAlert(context.Background(), nil, rule, event))
	require.Equal(t, 1, hit)

	// Second event within the window must be blocked by the rate limiter.
	event2 := &OpsAlertEvent{ID: 2, Status: "firing"}
	require.False(t, evaluator.maybeDispatchAlert(context.Background(), nil, rule, event2))
	require.Equal(t, 1, hit)
}

var _ = strings.TrimSpace // keep strings import if unused paths change
