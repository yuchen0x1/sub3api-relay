package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/apicompat"
)

// Manus exposes an asynchronous task API rather than a chat completion API, so
// an OpenAI-compatible request is served by creating a task and polling its
// event stream until the agent stops. Poll spacing stays well under the
// documented 100 req/min budget for task.listMessages.
const (
	manusDefaultChatModel   = "manus"
	manusPollInterval       = 2 * time.Second
	manusDefaultChatTimeout = 10 * time.Minute
	manusListMessagesPage   = 200
)

// ManusChatDelta receives assistant text as the agent produces it.
type ManusChatDelta func(text string)

// ManusChatCompletion is the terminal outcome of one OpenAI-compatible turn.
type ManusChatCompletion struct {
	TaskID string
	Text   string
}

// CreateChatTask converts an OpenAI Chat Completions payload into a Manus
// task.create call and returns the new task id.
func (s *ManusGatewayService) CreateChatTask(
	ctx context.Context,
	account *Account,
	req *apicompat.ChatCompletionsRequest,
) (string, error) {
	prompt := manusPromptFromMessages(req)
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("no message content to send to Manus")
	}
	payload, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"content": []map[string]string{{
				"type": "text",
				"text": prompt,
			}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal Manus task.create payload: %w", err)
	}
	body, err := s.callManusJSON(ctx, account, http.MethodPost, "task.create", "", payload)
	if err != nil {
		return "", err
	}
	taskID := gjson.GetBytes(body, "task_id").String()
	if taskID == "" {
		return "", errors.New("manus task.create returned no task_id")
	}
	return taskID, nil
}

// WaitForChatResult polls task.listMessages until the agent stops, fails, or
// asks for input, streaming assistant text through onDelta as it arrives.
func (s *ManusGatewayService) WaitForChatResult(
	ctx context.Context,
	account *Account,
	taskID string,
	onDelta ManusChatDelta,
) (*ManusChatCompletion, error) {
	deadline := time.Now().Add(manusDefaultChatTimeout)
	seen := make(map[string]struct{})
	cursor := ""
	var collected strings.Builder

	for {
		done, failure, err := s.drainManusEvents(ctx, account, taskID, &cursor, seen, &collected, onDelta)
		if err != nil {
			return nil, err
		}
		if failure != "" {
			return nil, errors.New(failure)
		}
		if done {
			return &ManusChatCompletion{TaskID: taskID, Text: collected.String()}, nil
		}
		if time.Now().After(deadline) {
			// The task keeps running upstream, so return the task id alongside
			// any partial text instead of discarding the work.
			return &ManusChatCompletion{TaskID: taskID, Text: collected.String()},
				fmt.Errorf("manus task %s did not finish within %s", taskID, manusDefaultChatTimeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(manusPollInterval):
		}
	}
}

// drainManusEvents consumes every task event page that is currently available.
func (s *ManusGatewayService) drainManusEvents(
	ctx context.Context,
	account *Account,
	taskID string,
	cursor *string,
	seen map[string]struct{},
	collected *strings.Builder,
	onDelta ManusChatDelta,
) (bool, string, error) {
	done := false
	for {
		query := url.Values{}
		query.Set("task_id", taskID)
		query.Set("order", "asc")
		query.Set("limit", strconv.Itoa(manusListMessagesPage))
		if *cursor != "" {
			query.Set("cursor", *cursor)
		}
		body, err := s.callManusJSON(ctx, account, http.MethodGet, "task.listMessages", query.Encode(), nil)
		if err != nil {
			return false, "", err
		}

		for _, event := range gjson.GetBytes(body, "messages").Array() {
			if id := event.Get("id").String(); id != "" {
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
			}
			switch event.Get("type").String() {
			case "assistant_message":
				text := event.Get("assistant_message.content").String()
				if text == "" {
					continue
				}
				if collected.Len() > 0 {
					text = "\n\n" + text
				}
				_, _ = collected.WriteString(text)
				if onDelta != nil {
					onDelta(text)
				}
			case "error_message":
				message := event.Get("error_message.content").String()
				if message == "" {
					message = "manus task failed"
				}
				return false, message, nil
			case "status_update":
				switch event.Get("status_update.agent_status").String() {
				case "stopped":
					done = true
				case "error":
					return false, "manus task ended with an error", nil
				case "waiting":
					// The agent wants a confirmation this bridge cannot answer,
					// so the turn ends with whatever it already produced.
					done = true
				}
			}
		}

		if gjson.GetBytes(body, "has_more").Bool() {
			if next := gjson.GetBytes(body, "next_cursor").String(); next != "" {
				*cursor = next
				continue
			}
		}
		return done, "", nil
	}
}

// callManusJSON performs one authenticated Manus call and returns its body,
// reusing Forward so proxy, TLS fingerprint and credential rules stay identical
// to the passthrough gateway.
func (s *ManusGatewayService) callManusJSON(
	ctx context.Context,
	account *Account,
	method string,
	endpoint string,
	rawQuery string,
	payload []byte,
) ([]byte, error) {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	var body io.Reader
	if payload != nil {
		headers.Set("Content-Type", "application/json")
		body = bytes.NewReader(payload)
	}
	resp, err := s.Forward(ctx, account, method, endpoint, rawQuery, headers, body)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read Manus %s response: %w", endpoint, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if message := gjson.GetBytes(raw, "error.message").String(); message != "" {
			return nil, fmt.Errorf("manus %s failed (%d): %s", endpoint, resp.StatusCode, message)
		}
		return nil, fmt.Errorf("manus %s failed (%d)", endpoint, resp.StatusCode)
	}
	if ok := gjson.GetBytes(raw, "ok"); ok.Exists() && !ok.Bool() {
		message := gjson.GetBytes(raw, "error.message").String()
		if message == "" {
			message = "unknown Manus error"
		}
		return nil, fmt.Errorf("manus %s failed: %s", endpoint, message)
	}
	return raw, nil
}

// manusPromptFromMessages flattens an OpenAI conversation into the single text
// prompt that task.create accepts.
func manusPromptFromMessages(req *apicompat.ChatCompletionsRequest) string {
	if req == nil {
		return ""
	}
	var sections []string
	if instructions := strings.TrimSpace(req.Instructions); instructions != "" {
		sections = append(sections, instructions)
	}
	for _, message := range req.Messages {
		text := strings.TrimSpace(manusTextFromContent(message.Content))
		if text == "" {
			continue
		}
		switch message.Role {
		case "assistant":
			sections = append(sections, "Assistant: "+text)
		case "tool", "function":
			sections = append(sections, "Tool result: "+text)
		default:
			sections = append(sections, text)
		}
	}
	return strings.Join(sections, "\n\n")
}

// manusTextFromContent accepts both the string and multi-part content shapes.
func manusTextFromContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	value := gjson.ParseBytes(raw)
	if value.Type == gjson.String {
		return value.String()
	}
	if !value.IsArray() {
		return ""
	}
	var parts []string
	for _, part := range value.Array() {
		switch part.Get("type").String() {
		case "text", "input_text", "output_text", "":
			if text := part.Get("text").String(); text != "" {
				parts = append(parts, text)
			}
		case "image_url":
			if imageURL := part.Get("image_url.url").String(); imageURL != "" {
				parts = append(parts, imageURL)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// ManusChatModelID resolves the model name echoed back to the client.
func ManusChatModelID(requested string) string {
	if trimmed := strings.TrimSpace(requested); trimmed != "" {
		return trimmed
	}
	return manusDefaultChatModel
}
