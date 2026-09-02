//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/xai"
)

// 上游地址不设白名单/内网/http 限制：任意自定义 base_url 均可构建端点。
func TestGrokAPIKeyURLPolicyAcceptsAnyUpstream(t *testing.T) {
	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "http://grok.example.test/v1",
		},
	}
	cfg := &config.Config{}

	responsesURL, err := buildGrokResponsesURL(account, cfg)
	require.NoError(t, err)
	require.Equal(t, "http://grok.example.test/v1/responses", responsesURL)

	chatURL, err := buildGrokChatCompletionsURL(account, cfg)
	require.NoError(t, err)
	require.Equal(t, "http://grok.example.test/v1/chat/completions", chatURL)

	mediaURL, err := buildGrokMediaURL(account, cfg, GrokMediaEndpointImagesGenerations, "")
	require.NoError(t, err)
	require.Equal(t, "http://grok.example.test/v1/images/generations", mediaURL)

	contentURL, err := buildGrokMediaURL(account, cfg, GrokMediaEndpointVideoContent, "request 123")
	require.NoError(t, err)
	require.Equal(t, "http://grok.example.test/v1/videos/request%20123/content", contentURL)

	// 内网/回环地址同样放行（自建网关场景）。
	account.Credentials["base_url"] = "https://127.0.0.1/v1"
	target, err := buildGrokResponsesURL(account, cfg)
	require.NoError(t, err)
	require.Equal(t, "https://127.0.0.1/v1/responses", target)
}

func TestGrokAPIKeyURLPolicyRedactsMalformedConfiguredURL(t *testing.T) {
	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://%zz:secret@grok.example.test/v1",
		},
	}
	cfg := &config.Config{}

	_, err := buildGrokResponsesURL(account, cfg)
	require.EqualError(t, err, "invalid base url: base URL rejected by URL security policy")
	require.NotContains(t, err.Error(), "secret")
}

func TestGrokOAuthURLPolicy(t *testing.T) {
	t.Run("default CLI gateway allowed", func(t *testing.T) {
		account := &Account{
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, xai.DefaultCLIBaseURL+"/responses", target)
	})

	t.Run("stored official API endpoint is honored (manual endpoint switch)", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": xai.DefaultBaseURL,
			},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, xai.DefaultBaseURL+"/responses", target)
	})

	t.Run("stored regional API endpoint is trusted", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": "https://us-west-2.api.x.ai/v1",
			},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, "https://us-west-2.api.x.ai/v1/responses", target)
	})

	t.Run("custom forwarding address follows operator policy", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": "https://relay.example.test/v1",
			},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, "https://relay.example.test/v1/responses", target)
	})

	t.Run("custom path prefix is preserved", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": "https://relay.example.test/xai/v1",
			},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, "https://relay.example.test/xai/v1/responses", target)
	})

	t.Run("insecure HTTP custom address is allowed", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": "http://relay.example.test/v1",
			},
		}
		cfg := &config.Config{}

		target, err := buildGrokResponsesURL(account, cfg)
		require.NoError(t, err)
		require.Equal(t, "http://relay.example.test/v1/responses", target)
	})
}

func TestGrokBillingURLFollowsAccountBaseURL(t *testing.T) {
	t.Run("oauth default stays on CLI gateway", func(t *testing.T) {
		account := &Account{
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{},
		}

		weeklyURL, err := buildGrokBillingURL(account, nil, true)
		require.NoError(t, err)
		require.Equal(t, xai.DefaultCLIBaseURL+"/billing?format=credits", weeklyURL)

		monthlyURL, err := buildGrokBillingURL(account, nil, false)
		require.NoError(t, err)
		require.Equal(t, xai.DefaultCLIBaseURL+"/billing", monthlyURL)
	})

	t.Run("oauth custom forwarding address carries billing probes", func(t *testing.T) {
		account := &Account{
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"base_url": "https://relay.example.test/v1",
			},
		}

		weeklyURL, err := buildGrokBillingURL(account, nil, true)
		require.NoError(t, err)
		require.Equal(t, "https://relay.example.test/v1/billing?format=credits", weeklyURL)
	})
}
