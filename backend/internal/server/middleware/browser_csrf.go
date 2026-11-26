package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/browserauth"
)

// BrowserSessionCSRF protects explicit browser-cookie mode. Legacy bearer
// clients are untouched because they do not send the mode header.
// A wildcard origin is intentionally rejected for cookie-authenticated writes.
func BrowserSessionCSRF(secret string, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" || origin == "*" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		if !browserauth.Requested(c.Request) || isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "BROWSER_ORIGIN_REQUIRED", "message": "Origin header is required"})
			return
		}
		if _, ok := allowed[origin]; !ok && !sameRequestOrigin(c, origin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "BROWSER_ORIGIN_FORBIDDEN", "message": "Origin is not allowed"})
			return
		}
		refreshToken := browserauth.RefreshToken(c.Request)
		if refreshToken != "" && !browserauth.VerifyCSRF(refreshToken, browserauth.CSRFToken(c.Request), c.GetHeader(browserauth.CSRFHeaderName), secret) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "BROWSER_CSRF_INVALID", "message": "CSRF validation failed"})
			return
		}
		c.Next()
	}
}

func sameRequestOrigin(c *gin.Context, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	return u.Scheme == scheme && strings.EqualFold(u.Host, c.Request.Host)
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
