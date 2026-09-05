package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// ForwardXSearch 承接独立 POST /x_search 端点：把搜索请求包装成带 x_search
// 工具的 Grok Responses 回合，上游 SSE 解析为 {output, results} JSON 返回。
//
// 返回值约定与 ForwardAlphaSearch 一致：仅当上游返回 2xx（一次真实成功的
// 搜索）时返回非 nil 的 *OpenAIForwardResult（WebSearchCalls=1，供按次计费）；
// 上游错误被原样透传给客户端时返回 (nil, nil)，不产生计费。
func (s *OpenAIGatewayService) ForwardXSearch(ctx context.Context, c *gin.Context, account *Account, body []byte, requestedModel string) (*OpenAIForwardResult, error) {
	if s == nil || c == nil || account == nil {
		return nil, fmt.Errorf("service, context, and account are required")
	}
	if account.Type != AccountTypeOAuth && account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("grok account type %s is not supported by x_search forwarding", account.Type)
	}

	upstreamModel := account.GetMappedModel(requestedModel)
	if strings.TrimSpace(upstreamModel) == "" {
		upstreamModel = grokDefaultResponsesModel
	}
	responsesBody, err := buildGrokXSearchResponsesBody(body, upstreamModel)
	if err != nil {
		return nil, err
	}

	token, _, err := s.getRequestCredential(ctx, c, account)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// x_search 是独立的工具端点，不参与会话粘性/缓存身份：cacheIdentity 留空。
	req, err := buildGrokResponsesRequest(ctx, c, account, responsesBody, token, "", s.cfg)
	if err != nil {
		return nil, err
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, true)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, fmt.Errorf("read x_search response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if upstreamMsg == "" {
			upstreamMsg = fmt.Sprintf("xAI upstream returned status %d", resp.StatusCode)
		}
		if s.shouldFailoverGrokUpstreamError(resp.StatusCode, respBody) {
			kind := "failover"
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  firstNonEmpty(resp.Header.Get("x-request-id"), resp.Header.Get("xai-request-id")),
				Kind:               kind,
				Message:            upstreamMsg,
			})
			s.handleGrokAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				ResponseHeaders:        resp.Header.Clone(),
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// 非 2xx（错误/重定向）原样透传给客户端：不是一次成功的搜索，不计费。
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		c.Data(resp.StatusCode, contentType, respBody)
		return nil, nil
	}

	s.updateGrokUsageFromResponse(ctx, account, resp.Header, resp.StatusCode)

	searchRespBody, err := openAIAlphaSearchResponseFromResponsesSSE(respBody)
	if err != nil {
		return nil, err
	}
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Data(http.StatusOK, "application/json", searchRespBody)
	return &OpenAIForwardResult{
		RequestID:      firstNonEmpty(resp.Header.Get("x-request-id"), resp.Header.Get("xai-request-id")),
		Model:          requestedModel,
		UpstreamModel:  upstreamModel,
		Duration:       time.Since(upstreamStart),
		WebSearchCalls: 1,
	}, nil
}

// buildGrokXSearchResponsesBody 把 /x_search 请求体包装为 Grok Responses 回合。
// 入站 schema 保持宽松：query 为搜索文本；settings/commands/input 整体序列化
// 进提示词（与 alpha search 的 PAT 兜底路径一致），未知字段不报错。
func buildGrokXSearchResponsesBody(body []byte, model string) ([]byte, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	payload := map[string]any{
		"model":  model,
		"stream": true,
		"store":  false,
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "input_text",
						"text": buildGrokXSearchPrompt(body),
					},
				},
			},
		},
		"tools": []any{map[string]any{"type": "x_search"}},
	}
	return json.Marshal(payload)
}

func buildGrokXSearchPrompt(body []byte) string {
	var b strings.Builder
	_, _ = b.WriteString("Execute this standalone X (Twitter) search request for another model.\n")
	_, _ = b.WriteString("Use the x_search tool to retrieve current posts matching the query.\n")
	_, _ = b.WriteString("Return concise results: post text, author handles, dates, and direct URLs when available.\n")
	if query := strings.TrimSpace(gjson.GetBytes(body, "query").String()); query != "" {
		_, _ = b.WriteString("\nQuery:\n")
		_, _ = b.WriteString(truncateOpenAIAlphaSearchPromptJSON(query, 4000))
	}
	if commands := strings.TrimSpace(gjson.GetBytes(body, "commands").Raw); commands != "" {
		_, _ = b.WriteString("\nCommands JSON:\n")
		_, _ = b.WriteString(truncateOpenAIAlphaSearchPromptJSON(commands, 12000))
	}
	if settings := strings.TrimSpace(gjson.GetBytes(body, "settings").Raw); settings != "" {
		_, _ = b.WriteString("\n\nSearch settings JSON:\n")
		_, _ = b.WriteString(truncateOpenAIAlphaSearchPromptJSON(settings, 4000))
	}
	if input := strings.TrimSpace(gjson.GetBytes(body, "input").Raw); input != "" {
		_, _ = b.WriteString("\n\nRecent conversation/input JSON:\n")
		_, _ = b.WriteString(truncateOpenAIAlphaSearchPromptJSON(input, 8000))
	}
	if b.Len() == 0 {
		return "Execute the requested X search and return concise source-backed results."
	}
	return b.String()
}
