//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/openai"
)

func TestCodexVersionConstants_Consistency(t *testing.T) {
	require.Equal(t, codexCLIVersion, openAICodexProbeVersion,
		"codexCLIVersion and openAICodexProbeVersion must stay in sync")

	require.True(t, strings.Contains(codexCLIUserAgent, openai.CodexDefaultOriginator+"/"+codexCLIVersion),
		"codexCLIUserAgent must embed codexCLIVersion")

	require.True(t, strings.Contains(DefaultOpenAICodexUserAgent, codexCLIVersion),
		"DefaultOpenAICodexUserAgent must embed codexCLIVersion")
}
