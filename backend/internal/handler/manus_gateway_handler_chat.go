package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/apicompat"
	"github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// ChatCompletions serves POST /v1/chat/completions for Manus platform groups.
// Manus has no chat completion endpoint of its own, so the request is mapped
// onto task.create plus task.listMessages polling and mapped back to the
// OpenAI response shape.
func (h *ManusGatewayHandler) ChatCompletions(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		writeManusChatError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeManusChatError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	var req apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeManusChatError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	account, release, ok := h.acquireManusAccount(c, apiKey.GroupID, writeManusChatError)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	model := service.ManusChatModelID(req.Model)
	if req.Stream {
		h.streamManusChatCompletion(c, account, &req, model)
		return
	}

	ctx := c.Request.Context()
	taskID, err := h.manusService.CreateChatTask(ctx, account, &req)
	if err != nil {
		writeManusChatError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	result, err := h.manusService.WaitForChatResult(ctx, account, taskID, nil)
	if err != nil {
		writeManusChatError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, apicompat.ChatCompletionsResponse{
		ID:      "chatcmpl-" + result.TaskID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []apicompat.ChatChoice{{
			Index: 0,
			Message: apicompat.ChatMessage{
				Role:    "assistant",
				Content: mustJSONString(result.Text),
			},
			FinishReason: "stop",
		}},
	})
}

// streamManusChatCompletion emits assistant text as OpenAI SSE chunks while the
// Manus agent works.
func (h *ManusGatewayHandler) streamManusChatCompletion(
	c *gin.Context,
	account *service.Account,
	req *apicompat.ChatCompletionsRequest,
	model string,
) {
	ctx := c.Request.Context()
	taskID, err := h.manusService.CreateChatTask(ctx, account, req)
	if err != nil {
		writeManusChatError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	id := "chatcmpl-" + taskID
	created := time.Now().Unix()
	emit := func(delta map[string]any, finishReason any) {
		chunk := map[string]any{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{{
				"index":         0,
				"delta":         delta,
				"finish_reason": finishReason,
			}},
		}
		payload, marshalErr := json.Marshal(chunk)
		if marshalErr != nil {
			return
		}
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		c.Writer.Flush()
	}

	emit(map[string]any{"role": "assistant"}, nil)
	_, waitErr := h.manusService.WaitForChatResult(ctx, account, taskID, func(text string) {
		emit(map[string]any{"content": text}, nil)
	})
	if waitErr != nil {
		// The stream already started, so the failure has to travel as content
		// rather than an HTTP status the client can no longer receive.
		emit(map[string]any{"content": "\n\n[manus error] " + waitErr.Error()}, nil)
	}
	emit(map[string]any{}, "stop")
	_, _ = io.WriteString(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// Models advertises the pseudo-model that maps to the Manus agent.
func (h *ManusGatewayHandler) Models(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data": []gin.H{{
			"id":       service.ManusChatModelID(""),
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "manus",
		}},
	})
}

func mustJSONString(value string) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return encoded
}

func writeManusChatError(c *gin.Context, status int, errType, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

// manusErrorWriter lets the passthrough and chat bridges report failures in
// their own protocol shape while sharing account selection.
type manusErrorWriter func(c *gin.Context, status int, code, message string)

// acquireManusAccount picks a Manus account for the group and reserves a
// concurrency slot, reporting failures through the caller's error writer.
func (h *ManusGatewayHandler) acquireManusAccount(
	c *gin.Context,
	groupID *int64,
	writeErr manusErrorWriter,
) (*service.Account, func(), bool) {
	selection, err := h.gatewayService.SelectAccountWithLoadAwareness(
		c.Request.Context(), groupID, "", "", nil, "", 0,
	)
	if err != nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		writeErr(c, http.StatusServiceUnavailable, "no_available_account", "No available Manus accounts")
		return nil, nil, false
	}
	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)

	release := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			writeErr(c, http.StatusServiceUnavailable, "no_available_account", "No available Manus accounts")
			return nil, nil, false
		}
		streamStarted := false
		release, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
			c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, false, &streamStarted,
		)
		if err != nil {
			writeErr(c, http.StatusTooManyRequests, "rate_limited", err.Error())
			return nil, nil, false
		}
	}
	return account, release, true
}
