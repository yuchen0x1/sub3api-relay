package service

import (
	"errors"
	"fmt"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/xai"
	"github.com/yuchen0x1/sub3api-relay/internal/util/urlvalidator"
)

func grokBaseURLValidator(account *Account, cfg *config.Config) (xai.BaseURLValidator, error) {
	if account == nil || !account.IsGrok() {
		return nil, fmt.Errorf("grok account is required")
	}
	switch account.Type {
	case AccountTypeOAuth:
		// Official gateway hosts are always trusted and always usable, even when
		// the operator enables a restrictive URL allowlist. A custom forwarding
		// host is vetted by the same operator policy as API-key accounts.
		//
		// The official-vs-custom decision is made on the host, not via
		// ValidateTrustedBaseURL: that validator relaxes to accept-any under the
		// XAI_ALLOW_UNSAFE_URL_OVERRIDES debug switch, which must never let an
		// OAuth bearer token reach an arbitrary custom host.
		policyValidator := grokOperatorPolicyValidator()
		return redactedGrokBaseURLValidator(func(raw string) (string, error) {
			if xai.IsOfficialBaseURL(raw) {
				return xai.ValidateTrustedBaseURL(raw)
			}
			return policyValidator(raw)
		}), nil
	case AccountTypeAPIKey:
		return redactedGrokBaseURLValidator(grokOperatorPolicyValidator()), nil
	default:
		return nil, fmt.Errorf("unsupported grok account type: %s", account.Type)
	}
}

// grokOperatorPolicyValidator 校验自定义 base_url 的基本格式（http/https 均可，
// 不再做域名白名单或内网限制）。
func grokOperatorPolicyValidator() xai.BaseURLValidator {
	return func(raw string) (string, error) {
		return urlvalidator.ValidateURLFormat(raw)
	}
}

func redactedGrokBaseURLValidator(validator xai.BaseURLValidator) xai.BaseURLValidator {
	return func(raw string) (string, error) {
		validated, err := validator(raw)
		if err != nil {
			return "", errors.New("base URL rejected by URL security policy")
		}
		return validated, nil
	}
}

func buildGrokResponsesURL(account *Account, cfg *config.Config) (string, error) {
	validator, err := grokBaseURLValidator(account, cfg)
	if err != nil {
		return "", err
	}
	return xai.BuildResponsesURLWithValidator(account.GetGrokBaseURL(), validator)
}

func buildGrokChatCompletionsURL(account *Account, cfg *config.Config) (string, error) {
	validator, err := grokBaseURLValidator(account, cfg)
	if err != nil {
		return "", err
	}
	return xai.BuildChatCompletionsURLWithValidator(account.GetGrokBaseURL(), validator)
}

// buildGrokBillingURL 解析 billing 探测端点：跟随账号的转发 base_url，
// 未定制的账号仍指向官方 CLI 网关。
func buildGrokBillingURL(account *Account, cfg *config.Config, weekly bool) (string, error) {
	validator, err := grokBaseURLValidator(account, cfg)
	if err != nil {
		return "", err
	}
	return xai.BuildBillingURLWithValidator(account.GetGrokBaseURL(), weekly, validator)
}

func buildGrokMediaURL(account *Account, cfg *config.Config, endpoint GrokMediaEndpoint, requestID string) (string, error) {
	validator, err := grokBaseURLValidator(account, cfg)
	if err != nil {
		return "", err
	}
	baseURL := account.GetGrokMediaBaseURL()
	switch endpoint {
	case GrokMediaEndpointImagesGenerations:
		return xai.BuildImagesGenerationsURLWithValidator(baseURL, validator)
	case GrokMediaEndpointImagesEdits:
		return xai.BuildImagesEditsURLWithValidator(baseURL, validator)
	case GrokMediaEndpointVideosGenerations:
		return xai.BuildVideosGenerationsURLWithValidator(baseURL, validator)
	case GrokMediaEndpointVideosEdits:
		return xai.BuildVideosEditsURLWithValidator(baseURL, validator)
	case GrokMediaEndpointVideosExtensions:
		return xai.BuildVideosExtensionsURLWithValidator(baseURL, validator)
	case GrokMediaEndpointVideoStatus:
		return xai.BuildVideoURLWithValidator(baseURL, requestID, validator)
	case GrokMediaEndpointVideoContent:
		videoURL, err := xai.BuildVideoURLWithValidator(baseURL, requestID, validator)
		if err != nil {
			return "", err
		}
		return videoURL + "/content", nil
	default:
		return "", fmt.Errorf("unsupported grok media endpoint: %s", endpoint)
	}
}
