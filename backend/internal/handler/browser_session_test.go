//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/browserauth"
	servermiddleware "github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type browserSessionRefreshCache struct {
	userHandlerRefreshTokenCacheStub
	data map[string]*service.RefreshTokenData
}

func (s *browserSessionRefreshCache) StoreRefreshToken(_ context.Context, hash string, data *service.RefreshTokenData, _ time.Duration) error {
	s.data[hash] = data
	return nil
}

func (s *browserSessionRefreshCache) GetRefreshToken(_ context.Context, hash string) (*service.RefreshTokenData, error) {
	data := s.data[hash]
	if data == nil {
		return nil, service.ErrRefreshTokenNotFound
	}
	return data, nil
}

func (s *browserSessionRefreshCache) DeleteRefreshToken(_ context.Context, hash string) error {
	delete(s.data, hash)
	return nil
}

func TestEstablishBrowserSessionRequiresMatchingTokenPair(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, subjectID := range []int64{7, 8} {
		name := "matching-user"
		if subjectID != 7 {
			name = "different-user"
		}
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{JWT: config.JWTConfig{Secret: "browser-session-test-secret", ExpireHour: 1, RefreshTokenExpireDays: 30}}
			user := &service.User{ID: 7, Email: "browser@example.com", Role: service.RoleUser, Status: service.StatusActive}
			cache := &browserSessionRefreshCache{data: make(map[string]*service.RefreshTokenData)}
			auth := service.NewAuthService(nil, &userHandlerRepoStub{user: user}, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
			pair, err := auth.GenerateTokenPair(context.Background(), user, "")
			require.NoError(t, err)
			h := &AuthHandler{cfg: cfg, authService: auth, settingSvc: service.NewSettingService(&oauthCaptchaSettingRepo{values: map[string]string{}}, cfg)}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/browser-session", strings.NewReader(`{"refresh_token":"`+pair.RefreshToken+`"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Request.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: subjectID})
			h.EstablishBrowserSession(c)
			if subjectID != user.ID {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Empty(t, w.Result().Cookies())
				return
			}
			require.Equal(t, http.StatusOK, w.Code)
			cookies := w.Result().Cookies()
			require.Len(t, cookies, 3)
			for _, cookie := range cookies {
				if cookie.Name == browserauth.RefreshCookieName {
					require.True(t, cookie.HttpOnly)
					require.NotEqual(t, http.SameSiteNoneMode, cookie.SameSite)
				}
			}
			var payload struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			require.NotEmpty(t, payload.Data["access_token"])
			require.Empty(t, payload.Data["refresh_token"])
			require.NotContains(t, w.Body.String(), pair.RefreshToken)
		})
	}
}

func TestBrowserRefreshLegacyMigrationAndCSRFGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name       string
		origin     string
		cookie     bool
		csrf       bool
		wantStatus int
	}{
		{name: "legacy bootstrap", origin: "http://app.example", wantStatus: http.StatusOK},
		{name: "bootstrap requires origin", wantStatus: http.StatusForbidden},
		{name: "bootstrap rejects hostile origin", origin: "http://evil.example", wantStatus: http.StatusForbidden},
		{name: "cookie cannot bypass csrf with body token", origin: "http://app.example", cookie: true, wantStatus: http.StatusForbidden},
		{name: "cookie refresh uses signed csrf", origin: "http://app.example", cookie: true, csrf: true, wantStatus: http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{JWT: config.JWTConfig{Secret: "browser-refresh-test-secret", ExpireHour: 1, RefreshTokenExpireDays: 30}}
			user := &service.User{ID: 7, Email: "browser@example.com", Role: service.RoleUser, Status: service.StatusActive}
			cache := &browserSessionRefreshCache{data: make(map[string]*service.RefreshTokenData)}
			auth := service.NewAuthService(nil, &userHandlerRepoStub{user: user}, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
			pair, err := auth.GenerateTokenPair(context.Background(), user, "")
			require.NoError(t, err)
			h := &AuthHandler{cfg: cfg, authService: auth, settingSvc: service.NewSettingService(&oauthCaptchaSettingRepo{values: map[string]string{}}, cfg)}
			router := gin.New()
			router.Use(servermiddleware.BrowserSessionCSRF(cfg.JWT.Secret, nil))
			router.POST("/api/v1/auth/refresh", h.RefreshToken)
			bodyRefresh := pair.RefreshToken
			if tt.cookie {
				bodyRefresh = "invalid-body-token"
			}
			req := httptest.NewRequest(http.MethodPost, "http://app.example/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"`+bodyRefresh+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.cookie {
				req.AddCookie(&http.Cookie{Name: browserauth.RefreshCookieName, Value: pair.RefreshToken})
			}
			if tt.csrf {
				csrf, err := browserauth.NewCSRFToken(pair.RefreshToken, cfg.JWT.Secret)
				require.NoError(t, err)
				req.AddCookie(&http.Cookie{Name: browserauth.CSRFCookieName, Value: csrf})
				req.Header.Set(browserauth.CSRFHeaderName, csrf)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusOK {
				require.Len(t, w.Result().Cookies(), 3)
				var payload struct {
					Data RefreshTokenResponse `json:"data"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
				require.NotEmpty(t, payload.Data.AccessToken)
				require.Empty(t, payload.Data.RefreshToken)
			} else {
				require.Empty(t, w.Result().Cookies())
			}
		})
	}
}

func TestBrowserAccessOnlyFallbackClearsPreviousSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "browser-fallback-test-secret", ExpireHour: 1}}
	user := &service.User{ID: 7, Email: "browser@example.com", Role: service.RoleUser, Status: service.StatusActive}
	// No refresh cache reproduces the access-only compatibility fallback.
	auth := service.NewAuthService(nil, &userHandlerRepoStub{user: user}, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	h := &AuthHandler{cfg: cfg, authService: auth}
	for _, tt := range []struct {
		name    string
		respond func(*gin.Context)
	}{
		{name: "password", respond: func(c *gin.Context) { h.respondWithTokenPair(c, user) }},
		{name: "passkey", respond: func(c *gin.Context) { respondWithTokenPair(c, auth, user) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			c.Request.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
			c.Request.AddCookie(&http.Cookie{Name: browserauth.RefreshCookieName, Value: "previous-user-refresh"})
			tt.respond(c)
			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, w.Result().Cookies(), 3)
			for _, cookie := range w.Result().Cookies() {
				require.Equal(t, -1, cookie.MaxAge)
			}
			var payload struct {
				Data AuthResponse `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			require.NotEmpty(t, payload.Data.AccessToken)
			require.Empty(t, payload.Data.RefreshToken)
		})
	}
}
