package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AlertNotificationMessage 是 ops 告警出口分发给所有通知 provider 的标准化载荷。
// 各 provider（邮件 / Bark / 通用 Webhook）自行决定如何渲染为具体格式。
type AlertNotificationMessage struct {
	Title       string
	Body        string
	Severity    string
	RuleName    string
	MetricType  string
	Operator    string
	Status      string
	EventID     int64
	MetricValue float64
	HasMetric   bool
	Threshold   float64
	FiredAt     time.Time
	Description string
}

// AlertNotificationProvider 把告警事件投递到某个外部通知渠道。
// 实现必须并发安全：告警评估器在单 goroutine 内顺序调用，但测试与
// 未来多 worker 场景不应假设单线程。
type AlertNotificationProvider interface {
	Name() string
	Send(ctx context.Context, msg AlertNotificationMessage) error
}

const (
	barkDefaultServerURL = "https://api.day.app"
	alertNotifyTimeout   = 10 * time.Second
)

// BarkAlertProvider 投递告警到 Bark（https://day.app）。
// JSON 形式调用 POST {server_url}/{key}，请求体包含 title/body/group，
// 避免标题正文含特殊字符时 URL 转义问题。
type BarkAlertProvider struct {
	serverURL string
	key       string
	client    *http.Client
}

// NewBarkAlertProvider 创建 Bark provider。
func NewBarkAlertProvider() *BarkAlertProvider {
	return &BarkAlertProvider{
		serverURL: barkDefaultServerURL,
		client: &http.Client{
			Timeout: alertNotifyTimeout,
		},
	}
}

func (p *BarkAlertProvider) Name() string {
	return "bark"
}

// SetConfig 更新 provider 配置（每次评估读取 DB 配置后调用）。
func (p *BarkAlertProvider) SetConfig(cfg OpsBarkNotificationConfig) {
	if p == nil {
		return
	}
	serverURL := strings.TrimSpace(cfg.ServerURL)
	if serverURL == "" {
		serverURL = barkDefaultServerURL
	}
	p.serverURL = strings.TrimRight(serverURL, "/")
	p.key = strings.TrimSpace(cfg.Key)
}

func (p *BarkAlertProvider) Send(ctx context.Context, msg AlertNotificationMessage) error {
	if p == nil || strings.TrimSpace(p.key) == "" {
		return fmt.Errorf("bark: key not configured")
	}
	endpoint := fmt.Sprintf("%s/%s", p.serverURL, url.PathEscape(strings.TrimSpace(p.key)))
	payload := map[string]any{
		"title": msg.Title,
		"body":  msg.Body,
		"group": "ops-alert",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bark: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("bark: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("bark: send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bark: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// WebhookAlertProvider 投递告警到任意 HTTP webhook，请求体为 JSON。
// 支持自定义 headers（例如钉钉 / 企业微信机器人的鉴权头）。
type WebhookAlertProvider struct {
	url     string
	headers map[string]string
	client  *http.Client
}

// NewWebhookAlertProvider 创建通用 Webhook provider。
func NewWebhookAlertProvider() *WebhookAlertProvider {
	return &WebhookAlertProvider{
		headers: map[string]string{},
		client: &http.Client{
			Timeout: alertNotifyTimeout,
		},
	}
}

func (p *WebhookAlertProvider) Name() string {
	return "webhook"
}

// SetConfig 更新 provider 配置。
func (p *WebhookAlertProvider) SetConfig(cfg OpsWebhookNotificationConfig) {
	if p == nil {
		return
	}
	p.url = strings.TrimSpace(cfg.URL)
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		key := strings.TrimSpace(k)
		if key != "" {
			headers[key] = strings.TrimSpace(v)
		}
	}
	p.headers = headers
}

func (p *WebhookAlertProvider) Send(ctx context.Context, msg AlertNotificationMessage) error {
	if p == nil || strings.TrimSpace(p.url) == "" {
		return fmt.Errorf("webhook: url not configured")
	}
	payload := map[string]any{
		"source":      "ops_alert",
		"title":       msg.Title,
		"body":        msg.Body,
		"severity":    msg.Severity,
		"rule_name":   msg.RuleName,
		"metric_type": msg.MetricType,
		"operator":    msg.Operator,
		"status":      msg.Status,
		"event_id":    msg.EventID,
		"fired_at":    msg.FiredAt.UTC().Format(time.RFC3339),
	}
	if msg.HasMetric {
		payload["metric_value"] = msg.MetricValue
		payload["threshold_value"] = msg.Threshold
	}
	if msg.Description != "" {
		payload["description"] = msg.Description
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// buildAlertNotificationMessage 由规则与事件构造标准化告警载荷。
func buildAlertNotificationMessage(rule *OpsAlertRule, event *OpsAlertEvent) AlertNotificationMessage {
	msg := AlertNotificationMessage{
		Severity:    strings.TrimSpace(rule.Severity),
		RuleName:    strings.TrimSpace(rule.Name),
		MetricType:  strings.TrimSpace(rule.MetricType),
		Operator:    strings.TrimSpace(rule.Operator),
		Status:      strings.TrimSpace(event.Status),
		EventID:     event.ID,
		Threshold:   rule.Threshold,
		Description: strings.TrimSpace(event.Description),
	}
	if !event.FiredAt.IsZero() {
		msg.FiredAt = event.FiredAt
	}
	if event.MetricValue != nil {
		msg.HasMetric = true
		msg.MetricValue = *event.MetricValue
	}
	if event.ThresholdValue != nil {
		msg.Threshold = *event.ThresholdValue
	}
	msg.Title = fmt.Sprintf("[Ops Alert][%s] %s", msg.Severity, msg.RuleName)
	msg.Body = buildOpsAlertPlainBody(rule, event)
	return msg
}

// buildOpsAlertPlainBody 构造纯文本告警正文（Bark / Webhook 使用）。
func buildOpsAlertPlainBody(rule *OpsAlertRule, event *OpsAlertEvent) string {
	if rule == nil || event == nil {
		return ""
	}
	metric := strings.TrimSpace(rule.MetricType)
	value := "-"
	threshold := fmt.Sprintf("%.2f", rule.Threshold)
	if event.MetricValue != nil {
		value = fmt.Sprintf("%.2f", *event.MetricValue)
	}
	if event.ThresholdValue != nil {
		threshold = fmt.Sprintf("%.2f", *event.ThresholdValue)
	}
	body := fmt.Sprintf("Rule: %s\nSeverity: %s\nStatus: %s\nMetric: %s %s %s (threshold %s)\nFired at: %s",
		strings.TrimSpace(rule.Name),
		strings.TrimSpace(rule.Severity),
		strings.TrimSpace(event.Status),
		metric,
		strings.TrimSpace(rule.Operator),
		value,
		threshold,
		event.FiredAt.UTC().Format(time.RFC3339),
	)
	if strings.TrimSpace(event.Description) != "" {
		body += "\nDescription: " + strings.TrimSpace(event.Description)
	}
	return body
}
