package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// grokStructuredErrorMessageCandidates 从错误响应体中提取候选错误消息：
// 依次尝试 error.message / error.error / 字符串化 error / message / detail，
// 非 JSON 响应体整体作为一个候选（便于匹配纯文本错误页）。
func grokStructuredErrorMessageCandidates(body []byte) []string {
	candidates := make([]string, 0, 6)
	appendCandidate := func(result gjson.Result) {
		if !result.Exists() {
			return
		}
		value := strings.TrimSpace(result.String())
		if value != "" {
			candidates = append(candidates, value)
		}
	}
	appendCandidate(gjson.GetBytes(body, "error.message"))
	appendCandidate(gjson.GetBytes(body, "error.error"))
	errorNode := gjson.GetBytes(body, "error")
	if errorNode.Type == gjson.String {
		appendCandidate(errorNode)
	}
	appendCandidate(gjson.GetBytes(body, "message"))
	appendCandidate(gjson.GetBytes(body, "detail"))
	if !json.Valid(body) {
		if plaintext := strings.TrimSpace(string(body)); plaintext != "" {
			candidates = append(candidates, plaintext)
		}
	}
	return candidates
}

func grokStringValue(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// isGrokCompactionReplayDecodeError 识别"压缩回放块解码失败"：xAI 会拒绝从其他
// 解码器/缓存上下文复制来的加密推理或 compaction 摘要（400/422 + decode/deserialize
// 与 compaction/summary/encrypted_content/response history 的组合信号）。
func isGrokCompactionReplayDecodeError(statusCode int, body []byte) bool {
	if (statusCode != http.StatusBadRequest && statusCode != http.StatusUnprocessableEntity) || len(body) == 0 {
		return false
	}
	for _, candidate := range grokStructuredErrorMessageCandidates(body) {
		message := strings.ToLower(candidate)
		decodeSignal := strings.Contains(message, "decode") ||
			strings.Contains(message, "deserialize") ||
			strings.Contains(message, "decoder")
		replaySignal := strings.Contains(message, "compaction") ||
			strings.Contains(message, "summary") ||
			strings.Contains(message, "encrypted_content") ||
			strings.Contains(message, "response history")
		if decodeSignal && replaySignal {
			return true
		}
	}
	return false
}

// sanitizeGrokCompactionReplayBody 为压缩回放重试清理请求体：保留可见摘要，
// 仅移除不透明的回放状态（加密推理、空 reasoning 项、悬空的 previous_response_id）。
func sanitizeGrokCompactionReplayBody(body []byte) ([]byte, bool, error) {
	converted, err := convertOpenAICompactInputsForGrok(body)
	if err != nil {
		return nil, false, fmt.Errorf("convert Grok compaction replay: %w", err)
	}
	var requestBody map[string]any
	decoder := json.NewDecoder(bytes.NewReader(converted))
	decoder.UseNumber()
	if err := decoder.Decode(&requestBody); err != nil {
		return nil, false, err
	}

	changed := !bytes.Equal(converted, body)
	if trimOpenAIEncryptedReasoningItems(requestBody) {
		changed = true
	}
	if dropEmptyGrokReplayReasoning(requestBody) {
		changed = true
	}
	if previousID, _ := requestBody["previous_response_id"].(string); strings.TrimSpace(previousID) != "" && !HasFunctionCallOutput(requestBody) {
		delete(requestBody, "previous_response_id")
		if _, exists := requestBody["store"]; !exists {
			requestBody["store"] = false
		}
		changed = true
	}
	if !changed {
		return body, false, nil
	}
	retryBody, err := marshalOpenAIUpstreamJSON(requestBody)
	if err != nil {
		return nil, false, err
	}
	return retryBody, true, nil
}

// dropEmptyGrokReplayReasoning 移除既无加密内容、也无可见 summary/content 的
// 空 reasoning 项——它们是回放状态被清理后留下的空壳，只会继续触发解码错误。
func dropEmptyGrokReplayReasoning(requestBody map[string]any) bool {
	items, ok := requestBody["input"].([]any)
	if !ok {
		return false
	}
	filtered := items[:0]
	changed := false
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok || strings.TrimSpace(grokStringValue(item["type"])) != "reasoning" {
			filtered = append(filtered, rawItem)
			continue
		}
		summary, _ := item["summary"].([]any)
		content, hasContent := item["content"]
		_, hasEncrypted := item["encrypted_content"]
		if hasEncrypted || len(summary) > 0 || (hasContent && content != nil) {
			filtered = append(filtered, rawItem)
			continue
		}
		changed = true
	}
	if changed {
		requestBody["input"] = filtered
	}
	return changed
}
