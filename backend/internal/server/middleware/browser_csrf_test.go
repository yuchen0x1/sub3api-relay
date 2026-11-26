package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/browserauth"
)

func TestBrowserSessionCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "test-secret"
	const refresh = "refresh-token"
	csrf, err := browserauth.NewCSRFToken(refresh, secret)
	require.NoError(t, err)

	router := gin.New()
	router.Use(BrowserSessionCSRF(secret, []string{"https://app.example"}))
	router.POST("/write", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	t.Run("legacy bearer bypasses browser gate", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/write", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	})
	t.Run("browser requires trusted origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/write", nil)
		req.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	})
	t.Run("browser validates csrf when refresh cookie exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/write", nil)
		req.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
		req.Header.Set("Origin", "https://app.example")
		req.AddCookie(&http.Cookie{Name: browserauth.RefreshCookieName, Value: refresh})
		req.AddCookie(&http.Cookie{Name: browserauth.CSRFCookieName, Value: csrf})
		req.Header.Set(browserauth.CSRFHeaderName, csrf)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)
	})
	t.Run("browser rejects missing csrf when refresh cookie exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/write", nil)
		req.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
		req.Header.Set("Origin", "https://app.example")
		req.AddCookie(&http.Cookie{Name: browserauth.RefreshCookieName, Value: refresh})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestBrowserSessionCSRFSameOriginWithoutCORSConfiguration(t *testing.T) {
	router := gin.New()
	router.Use(BrowserSessionCSRF("secret", nil))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tt := range []struct {
		origin string
		want   int
	}{
		{"http://app.example", http.StatusNoContent},
		{"http://evil.example", http.StatusForbidden},
		{"null", http.StatusForbidden},
		{"http://app.example/", http.StatusForbidden},
	} {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://app.example/login", nil)
			req.Header.Set(browserauth.SessionModeHeader, browserauth.SessionModeCookie)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tt.want, w.Code)
		})
	}
}
