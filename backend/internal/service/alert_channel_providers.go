package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// 本文件实现 Telegram / 企业微信群机器人 / Discord 三个即时通讯告警渠道。
// 均为一次性 HTTP POST，超时与 Bark / 通用 Webhook 一致（alertNotifyTimeout）。

// ============================== Telegram ==============================

// TelegramAlertProvider 通过 Telegram Bot API 的 sendMessage 推送告警。
type TelegramAlertProvider struct {
	botToken string
	chatID   string
	client   *http.Client
}

// NewTelegramAlertProvider 创建 Telegram provider。
func NewTelegramAlertProvider() *TelegramAlertProvider {
	return &TelegramAlertProvider{
		client: &http.Client{Timeout: alertNotifyTimeout},
	}
}

func (p *TelegramAlertProvider) Name() string { return "telegram" }

// SetConfig 更新 provider 配置（每次评估读取 DB 配置后调用）。
func (p *TelegramAlertProvider) SetConfig(cfg OpsTelegramNotificationConfig) {
	if p == nil {
		return
	}
	p.botToken = strings.TrimSpace(cfg.BotToken)
	p.chatID = strings.TrimSpace(cfg.ChatID)
}

func (p *TelegramAlertProvider) Send(ctx context.Context, msg AlertNotificationMessage) error {
	if p == nil || p.botToken == "" || p.chatID == "" {
		return fmt.Errorf("telegram: bot token or chat id not configured")
	}
	// sendMessage 文本模式（非 Markdown），避免告警正文中的特殊字符破坏解析。
	payload := map[string]any{
		"chat_id": p.chatID,
		"text":    fmt.Sprintf("%s\n%s", msg.Title, msg.Body),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ============================== 企业微信 ==============================

// WecomAlertProvider 通过企业微信群机器人 webhook 推送告警（text 消息）。
type WecomAlertProvider struct {
	webhookURL string
	client     *http.Client
}

// NewWecomAlertProvider 创建企业微信 provider。
func NewWecomAlertProvider() *WecomAlertProvider {
	return &WecomAlertProvider{
		client: &http.Client{Timeout: alertNotifyTimeout},
	}
}

func (p *WecomAlertProvider) Name() string { return "wecom" }

// SetConfig 更新 provider 配置。
func (p *WecomAlertProvider) SetConfig(cfg OpsWecomNotificationConfig) {
	if p == nil {
		return
	}
	p.webhookURL = strings.TrimSpace(cfg.WebhookURL)
}

func (p *WecomAlertProvider) Send(ctx context.Context, msg AlertNotificationMessage) error {
	if p == nil || p.webhookURL == "" {
		return fmt.Errorf("wecom: webhook url not configured")
	}
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content": fmt.Sprintf("%s\n%s", msg.Title, msg.Body),
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("wecom: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("wecom: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("wecom: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("wecom: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ============================== Discord ==============================

// DiscordAlertProvider 通过 Discord 频道 webhook 推送告警。
type DiscordAlertProvider struct {
	webhookURL string
	client     *http.Client
}

// NewDiscordAlertProvider 创建 Discord provider。
func NewDiscordAlertProvider() *DiscordAlertProvider {
	return &DiscordAlertProvider{
		client: &http.Client{Timeout: alertNotifyTimeout},
	}
}

func (p *DiscordAlertProvider) Name() string { return "discord" }

// SetConfig 更新 provider 配置。
func (p *DiscordAlertProvider) SetConfig(cfg OpsDiscordNotificationConfig) {
	if p == nil {
		return
	}
	p.webhookURL = strings.TrimSpace(cfg.WebhookURL)
}

func (p *DiscordAlertProvider) Send(ctx context.Context, msg AlertNotificationMessage) error {
	if p == nil || p.webhookURL == "" {
		return fmt.Errorf("discord: webhook url not configured")
	}
	payload := map[string]any{
		"content": fmt.Sprintf("**%s**\n%s", msg.Title, msg.Body),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("discord: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Discord webhook 成功时返回 204 No Content。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}
