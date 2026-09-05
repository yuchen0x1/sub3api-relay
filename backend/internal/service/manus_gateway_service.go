package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/util/responseheaders"
	"github.com/yuchen0x1/sub3api-relay/internal/util/urlvalidator"
)

var manusEndpointPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*\.[a-z][a-zA-Z0-9]*$`)

type ManusGatewayService struct {
	httpUpstream         HTTPUpstream
	cfg                  *config.Config
	tlsProfileService    *TLSFingerprintProfileService
	responseHeaderFilter *responseheaders.CompiledHeaderFilter
	staticProxyRouter    *StaticProxyRouter
}

func NewManusGatewayService(
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsProfileService *TLSFingerprintProfileService,
	// staticProxyRouter is optional (variadic); see NewOpenAIGatewayService.
	staticProxyRouters ...*StaticProxyRouter,
) *ManusGatewayService {
	headerCfg := config.ResponseHeaderConfig{}
	if cfg != nil {
		headerCfg = cfg.Security.ResponseHeaders
	}
	var staticProxyRouter *StaticProxyRouter
	for _, router := range staticProxyRouters {
		if router != nil {
			staticProxyRouter = router
			break
		}
	}
	return &ManusGatewayService{
		httpUpstream:         httpUpstream,
		cfg:                  cfg,
		tlsProfileService:    tlsProfileService,
		responseHeaderFilter: responseheaders.CompileHeaderFilter(headerCfg),
		staticProxyRouter:    staticProxyRouter,
	}
}

func (s *ManusGatewayService) Forward(
	ctx context.Context,
	account *Account,
	method string,
	endpoint string,
	rawQuery string,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, errors.New("manus upstream is not configured")
	}
	if account == nil || !account.IsManus() {
		return nil, errors.New("invalid Manus account")
	}

	// Static proxy routing (all-protocol coverage; see StaticProxyRouter).
	// The caller maps ErrStaticProxyCapacityUnavailable to a 429 response —
	// this service has no gin context, so it never writes HTTP bodies itself.
	staticRelease, staticErr := s.staticProxyRouter.Apply(ctx, account)
	if staticErr != nil {
		return nil, staticErr
	}
	if staticRelease != nil {
		defer staticRelease()
	}

	endpoint = strings.Trim(strings.TrimSpace(endpoint), "/")
	if !manusEndpointPattern.MatchString(endpoint) {
		return nil, errors.New("unsupported Manus endpoint")
	}

	baseURL, err := s.validateBaseURL(account.GetManusBaseURL())
	if err != nil {
		return nil, err
	}
	upstreamURL, err := buildManusURL(baseURL, endpoint, rawQuery)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, body)
	if err != nil {
		return nil, fmt.Errorf("build Manus request: %w", err)
	}
	copyManusRequestHeaders(req.Header, headers)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := account.GetManusAPIKey()
		if apiKey == "" {
			return nil, errors.New("manus API key is missing")
		}
		req.Header.Set("x-manus-api-key", apiKey)
	case AccountTypeOAuth:
		accessToken := account.GetManusAccessToken()
		if accessToken == "" {
			return nil, errors.New("manus access token is missing")
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
	default:
		return nil, fmt.Errorf("unsupported Manus account type: %s", account.Type)
	}
	account.ApplyHeaderOverrides(req.Header)
	// Account overrides must never turn Manus's mutually exclusive authentication
	// schemes into a request that carries both credentials.
	if account.Type == AccountTypeAPIKey {
		req.Header.Del("Authorization")
	} else {
		req.Header.Del("x-manus-api-key")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.tlsProfileService == nil {
		return s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	}
	return s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsProfileService.ResolveTLSProfile(account))
}

func (s *ManusGatewayService) WriteResponse(dst http.ResponseWriter, resp *http.Response) error {
	if resp == nil {
		return errors.New("empty Manus response")
	}
	responseheaders.WriteFilteredHeaders(dst.Header(), resp.Header, s.responseHeaderFilter)
	dst.WriteHeader(resp.StatusCode)
	_, err := io.Copy(dst, resp.Body)
	return err
}

func (s *ManusGatewayService) validateBaseURL(raw string) (string, error) {
	normalized, err := urlvalidator.ValidateURLFormat(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Manus base_url: %w", err)
	}
	return normalized, nil
}

func buildManusURL(baseURL, endpoint, rawQuery string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse Manus base_url: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/v2/" + endpoint
	u.RawPath = ""
	u.RawQuery = rawQuery
	u.Fragment = ""
	return u.String(), nil
}

func copyManusRequestHeaders(dst, src http.Header) {
	for _, name := range []string{"Accept", "Accept-Encoding", "Content-Type", "Content-Encoding", "User-Agent", "Idempotency-Key", "X-Request-Id"} {
		for _, value := range src.Values(name) {
			dst.Add(name, value)
		}
	}
	// These are intentionally omitted from the copy: Authorization,
	// x-manus-api-key, Cookie, Host, Content-Length, and hop-by-hop headers.
}
