package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaudeOAuthSystemPromptBlocksForModel(t *testing.T) {
	configured := `[{"type":"text","text":"{billing_header}"},{"type":"text","text":"{claude_code_system_prompt}"},{"type":"text","text":"{expansion}"}]`

	t.Run("fable models drop the CLI expansion block", func(t *testing.T) {
		for _, model := range []string{"claude-fable-5-1", "claude-fable-5.1-20260101", "Claude-Fable-5"} {
			got := claudeOAuthSystemPromptBlocksForModel(model, configured)
			var blocks []struct {
				Text string `json:"text"`
			}
			require.NoError(t, json.Unmarshal([]byte(got), &blocks))
			require.Len(t, blocks, 2)
			for _, b := range blocks {
				require.NotContains(t, b.Text, "expansion")
			}
		}
	})

	t.Run("non-fable models keep the configured blocks", func(t *testing.T) {
		require.Equal(t, configured, claudeOAuthSystemPromptBlocksForModel("claude-sonnet-4", configured))
		require.Equal(t, configured, claudeOAuthSystemPromptBlocksForModel("", configured))
	})
}
