package service

// Ops settings models stored in DB `settings` table (JSON blobs).

type OpsEmailNotificationConfig struct {
	Alert  OpsEmailAlertConfig  `json:"alert"`
	Report OpsEmailReportConfig `json:"report"`
	// Channels 额外通知渠道（Bark / 通用 Webhook），邮件仍由 Alert/Report 管理。
	Channels OpsNotificationChannelsConfig `json:"channels"`
}

type OpsEmailAlertConfig struct {
	Enabled               bool     `json:"enabled"`
	Recipients            []string `json:"recipients"`
	MinSeverity           string   `json:"min_severity"`
	RateLimitPerHour      int      `json:"rate_limit_per_hour"`
	BatchingWindowSeconds int      `json:"batching_window_seconds"`
	IncludeResolvedAlerts bool     `json:"include_resolved_alerts"`
}

type OpsEmailReportConfig struct {
	Enabled                         bool     `json:"enabled"`
	Recipients                      []string `json:"recipients"`
	DailySummaryEnabled             bool     `json:"daily_summary_enabled"`
	DailySummarySchedule            string   `json:"daily_summary_schedule"`
	WeeklySummaryEnabled            bool     `json:"weekly_summary_enabled"`
	WeeklySummarySchedule           string   `json:"weekly_summary_schedule"`
	ErrorDigestEnabled              bool     `json:"error_digest_enabled"`
	ErrorDigestSchedule             string   `json:"error_digest_schedule"`
	ErrorDigestMinCount             int      `json:"error_digest_min_count"`
	AccountHealthEnabled            bool     `json:"account_health_enabled"`
	AccountHealthSchedule           string   `json:"account_health_schedule"`
	AccountHealthErrorRateThreshold float64  `json:"account_health_error_rate_threshold"`
}

// OpsEmailNotificationConfigUpdateRequest allows partial updates, while the
// frontend can still send the full config shape.
type OpsEmailNotificationConfigUpdateRequest struct {
	Alert  *OpsEmailAlertConfig  `json:"alert"`
	Report *OpsEmailReportConfig `json:"report"`
	// Channels 允许部分更新额外通知渠道；为 nil 表示保持现状。
	Channels *OpsNotificationChannelsConfig `json:"channels"`
}

type OpsDistributedLockSettings struct {
	Enabled    bool   `json:"enabled"`
	Key        string `json:"key"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type OpsAlertSilenceEntry struct {
	RuleID     *int64   `json:"rule_id,omitempty"`
	Severities []string `json:"severities,omitempty"`

	UntilRFC3339 string `json:"until_rfc3339"`
	Reason       string `json:"reason"`
}

type OpsAlertSilencingSettings struct {
	Enabled bool `json:"enabled"`

	GlobalUntilRFC3339 string `json:"global_until_rfc3339"`
	GlobalReason       string `json:"global_reason"`

	Entries []OpsAlertSilenceEntry `json:"entries,omitempty"`
}

type OpsMetricThresholds struct {
	SLAPercentMin               *float64 `json:"sla_percent_min,omitempty"`                 // SLA低于此值变红
	TTFTp99MsMax                *float64 `json:"ttft_p99_ms_max,omitempty"`                 // TTFT P99高于此值变红
	RequestErrorRatePercentMax  *float64 `json:"request_error_rate_percent_max,omitempty"`  // 请求错误率高于此值变红
	UpstreamErrorRatePercentMax *float64 `json:"upstream_error_rate_percent_max,omitempty"` // 上游错误率高于此值变红
}

type OpsRuntimeLogConfig struct {
	Level           string         `json:"level"`
	EnableSampling  bool           `json:"enable_sampling"`
	SamplingInitial int            `json:"sampling_initial"`
	SamplingNext    int            `json:"sampling_thereafter"`
	Caller          bool           `json:"caller"`
	StacktraceLevel string         `json:"stacktrace_level"`
	RetentionDays   int            `json:"retention_days"`
	Source          string         `json:"source,omitempty"`
	UpdatedAt       string         `json:"updated_at,omitempty"`
	UpdatedByUserID int64          `json:"updated_by_user_id,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

type OpsAlertRuntimeSettings struct {
	EvaluationIntervalSeconds int `json:"evaluation_interval_seconds"`

	DistributedLock OpsDistributedLockSettings `json:"distributed_lock"`
	Silencing       OpsAlertSilencingSettings  `json:"silencing"`
	Thresholds      OpsMetricThresholds        `json:"thresholds"` // 指标阈值配置
}

// OpsAdvancedSettings stores advanced ops configuration (data retention, aggregation).
// OpsChannelMonitorAutoDisableSettings 渠道健康检查连续失败自动禁用配置（默认关闭）。
type OpsChannelMonitorAutoDisableSettings struct {
	Enabled             bool `json:"enabled"`
	ConsecutiveFailures int  `json:"consecutive_failures"`
	// AffectsAccounts 为 true 时，连续失败还会把同 Provider 的可调度账号
	// 临时标记为不可调度（temp_unschedulable_until），实现"监控即路由健康"闭环。
	AffectsAccounts bool `json:"affects_accounts"`
}

type OpsAdvancedSettings struct {
	DataRetention               OpsDataRetentionSettings               `json:"data_retention"`
	ChannelMonitorAutoDisable   OpsChannelMonitorAutoDisableSettings   `json:"channel_monitor_auto_disable"`
	Aggregation                 OpsAggregationSettings                 `json:"aggregation"`
	OpenAIAccountQuotaAutoPause OpsOpenAIAccountQuotaAutoPauseSettings `json:"openai_account_quota_auto_pause"`
	IgnoreCountTokensErrors     bool                                   `json:"ignore_count_tokens_errors"`
	IgnoreContextCanceled       bool                                   `json:"ignore_context_canceled"`
	IgnoreNoAvailableAccounts   bool                                   `json:"ignore_no_available_accounts"`
	// Deprecated compatibility field. It is always normalized to true.
	IgnoreInvalidApiKeyErrors       bool `json:"ignore_invalid_api_key_errors"`
	IgnoreInsufficientBalanceErrors bool `json:"ignore_insufficient_balance_errors"`
	DisplayOpenAITokenStats         bool `json:"display_openai_token_stats"`
	DisplayAlertEvents              bool `json:"display_alert_events"`
	AutoRefreshEnabled              bool `json:"auto_refresh_enabled"`
	AutoRefreshIntervalSec          int  `json:"auto_refresh_interval_seconds"`
}

type OpsOpenAIAccountQuotaAutoPauseSettings struct {
	DefaultThreshold5h float64 `json:"default_threshold_5h"`
	DefaultThreshold7d float64 `json:"default_threshold_7d"`
}

type OpsDataRetentionSettings struct {
	CleanupEnabled             bool   `json:"cleanup_enabled"`
	CleanupSchedule            string `json:"cleanup_schedule"`
	ErrorLogRetentionDays      int    `json:"error_log_retention_days"`
	MinuteMetricsRetentionDays int    `json:"minute_metrics_retention_days"`
	HourlyMetricsRetentionDays int    `json:"hourly_metrics_retention_days"`
}

type OpsAggregationSettings struct {
	AggregationEnabled bool `json:"aggregation_enabled"`
}

// OpsNotificationChannelsConfig 告警事件在邮件之外的补充通知渠道。
type OpsNotificationChannelsConfig struct {
	Bark     OpsBarkNotificationConfig     `json:"bark"`
	Webhook  OpsWebhookNotificationConfig  `json:"webhook"`
	Telegram OpsTelegramNotificationConfig `json:"telegram"`
	Wecom    OpsWecomNotificationConfig    `json:"wecom"`
	Discord  OpsDiscordNotificationConfig  `json:"discord"`
}

// OpsTelegramNotificationConfig Telegram Bot 推送配置。
// 通过 Bot API 的 sendMessage 以纯文本推送告警。
type OpsTelegramNotificationConfig struct {
	Enabled          bool   `json:"enabled"`
	BotToken         string `json:"bot_token"`
	ChatID           string `json:"chat_id"`
	MinSeverity      string `json:"min_severity"`
	RateLimitPerHour int    `json:"rate_limit_per_hour"`
}

// OpsWecomNotificationConfig 企业微信群机器人配置。
// WebhookURL 为群机器人 webhook 地址（https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...）。
type OpsWecomNotificationConfig struct {
	Enabled          bool   `json:"enabled"`
	WebhookURL       string `json:"webhook_url"`
	MinSeverity      string `json:"min_severity"`
	RateLimitPerHour int    `json:"rate_limit_per_hour"`
}

// OpsDiscordNotificationConfig Discord Webhook 配置。
// WebhookURL 为频道 webhook 地址（https://discord.com/api/webhooks/...）。
type OpsDiscordNotificationConfig struct {
	Enabled          bool   `json:"enabled"`
	WebhookURL       string `json:"webhook_url"`
	MinSeverity      string `json:"min_severity"`
	RateLimitPerHour int    `json:"rate_limit_per_hour"`
}

// OpsBarkNotificationConfig Bark 推送配置（https://day.app）。
// ServerURL 为空时使用官方 https://api.day.app。
type OpsBarkNotificationConfig struct {
	Enabled          bool   `json:"enabled"`
	ServerURL        string `json:"server_url"`
	Key              string `json:"key"`
	MinSeverity      string `json:"min_severity"`
	RateLimitPerHour int    `json:"rate_limit_per_hour"`
}

// OpsWebhookNotificationConfig 通用 Webhook 配置。
type OpsWebhookNotificationConfig struct {
	Enabled          bool              `json:"enabled"`
	URL              string            `json:"url"`
	Headers          map[string]string `json:"headers,omitempty"`
	MinSeverity      string            `json:"min_severity"`
	RateLimitPerHour int               `json:"rate_limit_per_hour"`
}
