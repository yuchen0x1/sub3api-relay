package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/apicompat"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestManusChatCompletionResponseMatchesOpenAIShape(t *testing.T) {
	resp := apicompat.ChatCompletionsResponse{
		ID:      "chatcmpl-tsk_123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   service.ManusChatModelID(""),
		Choices: []apicompat.ChatChoice{{
			Index: 0,
			Message: apicompat.ChatMessage{
				Role:    "assistant",
				Content: mustJSONString("hi \"there\"\nsecond line"),
			},
			FinishReason: "stop",
		}},
	}

	encoded, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	require.Equal(t, "chat.completion", decoded.Object)
	require.Equal(t, "manus", decoded.Model)
	require.Len(t, decoded.Choices, 1)
	require.Equal(t, "assistant", decoded.Choices[0].Message.Role)
	require.Equal(t, "hi \"there\"\nsecond line", decoded.Choices[0].Message.Content)
	require.Equal(t, "stop", decoded.Choices[0].FinishReason)
}

func TestManusChatModelIDEchoesRequestedModel(t *testing.T) {
	require.Equal(t, "manus", service.ManusChatModelID(""))
	require.Equal(t, "manus", service.ManusChatModelID("   "))
	require.Equal(t, "gpt-4o", service.ManusChatModelID("gpt-4o"))
}
