package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/apicompat"
	"github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// Messages serves POST /v1/messages for Manus platform groups. Many Claude
// clients call Anthropic Messages rather than OpenAI Chat Completions; without
// this bridge a Manus group can pass account tests but fail real user traffic.
func (h *ManusGatewayHandler) Messages(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		writeManusAnthropicError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeManusAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	var anthropicReq apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		writeManusAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	model := strings.TrimSpace(anthropicReq.Model)
	if model == "" {
		writeManusAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	chatReq, err := apicompat.AnthropicToChatCompletionsRequest(&anthropicReq)
	if err != nil {
		writeManusAnthropicError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	account, release, ok := h.acquireManusAccount(c, apiKey.GroupID, writeManusAnthropicError)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	if anthropicReq.Stream {
		h.streamManusAnthropicMessage(c, account, chatReq, model)
		return
	}

	ctx := c.Request.Context()
	taskID, err := h.manusService.CreateChatTask(ctx, account, chatReq)
	if err != nil {
		writeManusAnthropicError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	result, err := h.manusService.WaitForChatResult(ctx, account, taskID, nil)
	if err != nil {
		writeManusAnthropicError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, manusAnthropicResponse(result.TaskID, result.Text, model))
}

func (h *ManusGatewayHandler) streamManusAnthropicMessage(
	c *gin.Context,
	account *service.Account,
	req *apicompat.ChatCompletionsRequest,
	model string,
) {
	ctx := c.Request.Context()
	taskID, err := h.manusService.CreateChatTask(ctx, account, req)
	if err != nil {
		writeManusAnthropicError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	messageID := "msg-" + taskID
	blockIndex := 0
	blockStarted := false
	emit := func(event string, payload any) {
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return
		}
		_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", encoded)
		c.Writer.Flush()
	}

	emit("message_start", apicompat.AnthropicStreamEvent{
		Type: "message_start",
		Message: &apicompat.AnthropicResponse{
			ID:         messageID,
			Type:       "message",
			Role:       "assistant",
			Content:    []apicompat.AnthropicContentBlock{},
			Model:      model,
			StopReason: nil,
			Usage:      apicompat.AnthropicUsage{},
		},
	})

	_, waitErr := h.manusService.WaitForChatResult(ctx, account, taskID, func(text string) {
		if text == "" {
			return
		}
		if !blockStarted {
			blockStarted = true
			emit("content_block_start", apicompat.AnthropicStreamEvent{
				Type:         "content_block_start",
				Index:        &blockIndex,
				ContentBlock: &apicompat.AnthropicContentBlock{Type: "text", Text: ""},
			})
		}
		emit("content_block_delta", apicompat.AnthropicStreamEvent{
			Type:  "content_block_delta",
			Index: &blockIndex,
			Delta: &apicompat.AnthropicDelta{Type: "text_delta", Text: text},
		})
	})
	if waitErr != nil {
		if !blockStarted {
			blockStarted = true
			emit("content_block_start", apicompat.AnthropicStreamEvent{
				Type:         "content_block_start",
				Index:        &blockIndex,
				ContentBlock: &apicompat.AnthropicContentBlock{Type: "text", Text: ""},
			})
		}
		emit("content_block_delta", apicompat.AnthropicStreamEvent{
			Type:  "content_block_delta",
			Index: &blockIndex,
			Delta: &apicompat.AnthropicDelta{Type: "text_delta", Text: "\n\n[manus error] " + waitErr.Error()},
		})
	}
	if blockStarted {
		emit("content_block_stop", apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &blockIndex})
	}
	emit("message_delta", apicompat.AnthropicStreamEvent{
		Type:  "message_delta",
		Delta: &apicompat.AnthropicDelta{StopReason: "end_turn", StopSequence: nil},
		Usage: &apicompat.AnthropicUsage{},
	})
	emit("message_stop", apicompat.AnthropicStreamEvent{Type: "message_stop"})
}

func manusAnthropicResponse(taskID, text, model string) apicompat.AnthropicResponse {
	stopReason := "end_turn"
	return apicompat.AnthropicResponse{
		ID:   "msg-" + taskID,
		Type: "message",
		Role: "assistant",
		Content: []apicompat.AnthropicContentBlock{{
			Type: "text",
			Text: text,
		}},
		Model:      model,
		StopReason: &stopReason,
		Usage:      apicompat.AnthropicUsage{},
	}
}

func writeManusAnthropicError(c *gin.Context, status int, errType, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}
