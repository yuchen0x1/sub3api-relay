package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/httpclient"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type githubReleaseClient struct {
	httpClient         *http.Client
	downloadHTTPClient *http.Client
	updateGitHubToken  string
}

type githubReleaseClientError struct {
	err error
}

var githubReleaseTagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
var githubReleaseAssetHrefPattern = regexp.MustCompile(`href="([^"]*/releases/download/[^\"]+)"`)

// NewGitHubReleaseClient 创建 GitHub Release 客户端
// proxyURL 为空时直连 GitHub，支持 http/https/socks5/socks5h 协议
// 代理配置失败时行为由 allowDirectOnProxyError 控制：
//   - false（默认）：返回错误占位客户端，禁止回退到直连
//   - true：回退到直连（仅限管理员显式开启）
func NewGitHubReleaseClient(proxyURL string, allowDirectOnProxyError bool) service.GitHubReleaseClient {
	// 安全说明：httpclient.GetClient 的错误链（url.Parse / proxyutil）不含明文代理凭据，
	// 但仍通过 slog 仅在服务端日志记录，不会暴露给 HTTP 响应。
	sharedClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  30 * time.Second,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		sharedClient = &http.Client{Timeout: 30 * time.Second}
	}
	apiClient := cloneHTTPClient(sharedClient)
	apiClient.CheckRedirect = githubAPICheckRedirect(apiClient.CheckRedirect)

	// 下载客户端需要更长的超时时间
	downloadClient, err := httpclient.GetClient(httpclient.Options{
		Timeout:  10 * time.Minute,
		ProxyURL: proxyURL,
	})
	if err != nil {
		if strings.TrimSpace(proxyURL) != "" && !allowDirectOnProxyError {
			slog.Warn("proxy download client init failed, all requests will fail", "service", "github_release", "error", err)
			return &githubReleaseClientError{err: fmt.Errorf("proxy client init failed and direct fallback is disabled; set security.proxy_fallback.allow_direct_on_error=true to allow fallback: %w", err)}
		}
		downloadClient = &http.Client{Timeout: 10 * time.Minute}
	}
	downloadClient = cloneHTTPClient(downloadClient)

	return &githubReleaseClient{
		httpClient:         apiClient,
		downloadHTTPClient: downloadClient,
		updateGitHubToken:  os.Getenv("UPDATE_GITHUB_TOKEN"),
	}
}

func cloneHTTPClient(client *http.Client) *http.Client {
	cloned := *client
	return &cloned
}

func isGitHubAPIURL(url *url.URL) bool {
	return url != nil && strings.EqualFold(url.Scheme, "https") && url.User == nil &&
		strings.EqualFold(url.Host, "api.github.com")
}

func githubAPICheckRedirect(previous func(*http.Request, []*http.Request) error) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !isGitHubAPIURL(req.URL) {
			req.Header.Del("Authorization")
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
}

func (c *githubReleaseClient) newAPIRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Sub3API-Updater")
	if c.updateGitHubToken != "" && isGitHubAPIURL(req.URL) {
		req.Header.Set("Authorization", "Bearer "+c.updateGitHubToken)
	}
	return req, nil
}

func (c *githubReleaseClientError) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	return nil, c.err
}

func (c *githubReleaseClientError) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	return c.err
}

func (c *githubReleaseClientError) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	return nil, c.err
}

// FetchLatestRelease fetches a release from GitHub.
func (c *githubReleaseClient) FetchLatestRelease(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	return c.fetchLatestReleaseFromGitHub(ctx, repo)
}

func (c *githubReleaseClient) fetchLatestReleaseFromGitHub(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			if release, fallbackErr := c.fetchLatestReleaseFromGitHubWeb(ctx, repo); fallbackErr == nil {
				return release, nil
			} else {
				return nil, fmt.Errorf("GitHub API returned %d; web fallback failed: %w", resp.StatusCode, fallbackErr)
			}
		}
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// fetchLatestReleaseFromGitHubWeb is a rate-limit fallback for environments
// where api.github.com is exhausted or filtered while github.com remains
// reachable. The release page exposes the canonical tag redirect and the
// expanded-assets fragment contains the same download URLs needed by the
// updater.
func (c *githubReleaseClient) fetchLatestReleaseFromGitHubWeb(ctx context.Context, repo string) (*service.GitHubRelease, error) {
	latestURL := fmt.Sprintf("https://github.com/%s/releases/latest", strings.Trim(repo, "/"))
	req, err := c.newAPIRequest(ctx, latestURL)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub release page returned %d", resp.StatusCode)
	}

	tag, err := releaseTagFromGitHubURL(resp.Request.URL, repo)
	if err != nil {
		return nil, err
	}

	release := &service.GitHubRelease{
		TagName:    tag,
		Name:       "Sub3API " + strings.TrimPrefix(tag, "v"),
		HTMLURL:    fmt.Sprintf("https://github.com/%s/releases/tag/%s", strings.Trim(repo, "/"), tag),
		Draft:      false,
		Prerelease: false,
	}

	assets, assetsErr := c.fetchReleaseAssetsFromGitHubWeb(ctx, repo, tag)
	if assetsErr != nil {
		slog.Warn("GitHub release asset fallback failed; version check will continue without assets", "service", "github_release", "repo", repo, "tag", tag, "error", assetsErr)
	} else {
		release.Assets = assets
	}
	return release, nil
}

func releaseTagFromGitHubURL(releaseURL *url.URL, repo string) (string, error) {
	if releaseURL == nil || !strings.EqualFold(releaseURL.Scheme, "https") || !strings.EqualFold(releaseURL.Host, "github.com") {
		return "", fmt.Errorf("GitHub latest release redirect did not resolve to github.com")
	}
	expectedPrefix := "/" + strings.Trim(repo, "/") + "/releases/tag/"
	if !strings.HasPrefix(releaseURL.Path, expectedPrefix) {
		return "", fmt.Errorf("GitHub latest release redirect resolved to an unexpected path")
	}
	tag, err := url.PathUnescape(strings.TrimPrefix(releaseURL.Path, expectedPrefix))
	if err != nil || !githubReleaseTagPattern.MatchString(tag) {
		return "", fmt.Errorf("GitHub latest release redirect returned invalid tag")
	}
	return tag, nil
}

func (c *githubReleaseClient) fetchReleaseAssetsFromGitHubWeb(ctx context.Context, repo, tag string) ([]service.GitHubAsset, error) {
	assetsURL := fmt.Sprintf("https://github.com/%s/releases/expanded_assets/%s", strings.Trim(repo, "/"), url.PathEscape(tag))
	req, err := c.newAPIRequest(ctx, assetsURL)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub expanded assets page returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	baseURL := "https://github.com"
	seen := make(map[string]struct{})
	assets := make([]service.GitHubAsset, 0)
	for _, match := range githubReleaseAssetHrefPattern.FindAllSubmatch(body, -1) {
		href := html.UnescapeString(string(match[1]))
		parsed, err := url.Parse(href)
		if err != nil {
			continue
		}
		if parsed.IsAbs() {
			if !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Host, "github.com") {
				continue
			}
		} else {
			parsed.Scheme = "https"
			parsed.Host = "github.com"
		}
		name := path.Base(parsed.Path)
		if name == "." || name == "/" || strings.TrimSpace(name) == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		assets = append(assets, service.GitHubAsset{
			Name:               name,
			BrowserDownloadURL: baseURL + parsed.EscapedPath(),
		})
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("GitHub expanded assets page contained no release assets")
	}
	return assets, nil
}

// FetchRecentReleases fetches recent GitHub releases for rollback choices.
func (c *githubReleaseClient) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return c.fetchRecentReleasesFromGitHub(ctx, repo, perPage)
}

func (c *githubReleaseClient) fetchRecentReleasesFromGitHub(ctx context.Context, repo string, perPage int) ([]*service.GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", repo, perPage)

	req, err := c.newAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []*service.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (c *githubReleaseClient) DownloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// 使用预配置的下载客户端（已包含代理配置）
	resp, err := c.downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// SECURITY: Check Content-Length if available
	if resp.ContentLength > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, maxSize)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	// SECURITY: Use LimitReader to enforce max download size even if Content-Length is missing/wrong
	limited := io.LimitReader(resp.Body, maxSize+1)
	written, err := io.Copy(out, limited)

	// Close file before attempting to remove (required on Windows)
	_ = out.Close()

	if err != nil {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return err
	}

	// Check if we hit the limit (downloaded more than maxSize)
	if written > maxSize {
		_ = os.Remove(dest) // Clean up partial file (best-effort)
		return fmt.Errorf("download exceeded maximum size of %d bytes", maxSize)
	}

	return nil
}

func (c *githubReleaseClient) FetchChecksumFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
