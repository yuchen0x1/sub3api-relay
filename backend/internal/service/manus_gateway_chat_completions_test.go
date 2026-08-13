package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/apicompat"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/tlsfingerprint"
)

// fakeManusUpstream emulates the Manus v2 task API: task.create returns a task
// id and the first task.listMessages poll reports the agent still running.
type fakeManusUpstream struct {
	polls       atomic.Int32
	createBody  string
	lastListURL string
}

func (f *fakeManusUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	respond := func(payload string) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(payload)),
		}, nil
	}
	switch {
	case strings.Contains(req.URL.Path, "task.create"):
		body, _ := io.ReadAll(req.Body)
		f.createBody = string(body)
		return respond(`{"ok":true,"task_id":"tsk_123","task_title":"t"}`)
	case strings.Contains(req.URL.Path, "task.listMessages"):
		f.lastListURL = req.URL.String()
		if f.polls.Add(1) == 1 {
			return respond(`{"ok":true,"messages":[
				{"id":"e1","type":"status_update","status_update":{"agent_status":"running"}}
			],"has_more":false}`)
		}
		return respond(`{"ok":true,"messages":[
			{"id":"e2","type":"assistant_message","assistant_message":{"content":"hello from manus"}},
			{"id":"e3","type":"status_update","status_update":{"agent_status":"stopped"}}
		],"has_more":false}`)
	}
	return respond(`{"ok":true}`)
}

func (f *fakeManusUpstream) DoWithTLS(req *http.Request, p string, id int64, c int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return f.Do(req, p, id, c)
}

func newManusChatTestService(upstream HTTPUpstream) (*ManusGatewayService, *Account) {
	svc := NewManusGatewayService(upstream, &config.Config{
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true}},
	}, nil)
	account := &Account{
		ID: 7, Platform: PlatformManus, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "http://manus.test", "api_key": "server-key"},
	}
	return svc, account
}

func TestManusChatBridgeCreatesTaskAndPollsUntilStopped(t *testing.T) {
	upstream := &fakeManusUpstream{}
	svc, account := newManusChatTestService(upstream)

	req := &apicompat.ChatCompletionsRequest{
		Model: "manus",
		Messages: []apicompat.ChatMessage{
			{Role: "system", Content: json.RawMessage(`"be brief"`)},
			{Role: "user", Content: json.RawMessage(`"say hi"`)},
		},
	}

	taskID, err := svc.CreateChatTask(t.Context(), account, req)
	require.NoError(t, err)
	require.Equal(t, "tsk_123", taskID)
	require.JSONEq(t, `{"message":{"content":"be brief\n\nsay hi"}}`, upstream.createBody)

	var deltas []string
	result, err := svc.WaitForChatResult(t.Context(), account, taskID, func(text string) {
		deltas = append(deltas, text)
	})
	require.NoError(t, err)
	require.Equal(t, "hello from manus", result.Text)
	require.Equal(t, []string{"hello from manus"}, deltas)
	require.GreaterOrEqual(t, upstream.polls.Load(), int32(2))
	require.Contains(t, upstream.lastListURL, "task_id=tsk_123")
	require.Contains(t, upstream.lastListURL, "order=asc")
}

func TestManusChatBridgeSurfacesTaskError(t *testing.T) {
	upstream := &manusErrorUpstream{}
	svc, account := newManusChatTestService(upstream)

	_, err := svc.WaitForChatResult(t.Context(), account, "tsk_err", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "quota exhausted")
}

type manusErrorUpstream struct{}

func (m *manusErrorUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"ok":true,"messages":[
			{"id":"e1","type":"error_message","error_message":{"content":"quota exhausted"}}
		],"has_more":false}`)),
	}, nil
}

func (m *manusErrorUpstream) DoWithTLS(req *http.Request, p string, id int64, c int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return m.Do(req, p, id, c)
}

func TestManusPromptFlattensMultiPartContent(t *testing.T) {
	prompt := manusPromptFromMessages(&apicompat.ChatCompletionsRequest{
		Messages: []apicompat.ChatMessage{
			{Role: "user", Content: json.RawMessage(`[{"type":"text","text":"part one"},{"type":"text","text":"part two"}]`)},
			{Role: "assistant", Content: json.RawMessage(`"earlier reply"`)},
		},
	})
	require.Equal(t, "part one\npart two\n\nAssistant: earlier reply", prompt)
}
