package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/browserauth"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	servermiddleware "github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
)

const (
	BrowserSessionModeHeader = browserauth.SessionModeHeader
	BrowserSessionModeCookie = browserauth.SessionModeCookie
	BrowserRefreshCookieName = browserauth.RefreshCookieName
	BrowserCSRFCookieName    = browserauth.CSRFCookieName
	BrowserCSRFHeaderName    = browserauth.CSRFHeaderName
)

func BrowserSessionRequested(c *gin.Context) bool {
	return c != nil && browserauth.Requested(c.Request)
}

func BrowserRefreshToken(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return browserauth.RefreshToken(c.Request)
}

func SetBrowserSessionCookies(c *gin.Context, refreshToken, secret string, maxAge int) error {
	if c == nil {
		return browserauth.ErrInvalidCSRF
	}
	return browserauth.SetCookies(c.Writer, refreshToken, secret, maxAge, requestHTTPS(c), http.SameSiteLaxMode)
}

func ClearBrowserSessionCookies(c *gin.Context) {
	if c == nil {
		return
	}
	browserauth.ClearCookies(c.Writer, requestHTTPS(c), http.SameSiteLaxMode)
}

func VerifyBrowserCSRF(c *gin.Context, refreshToken, secret string) bool {
	if c == nil {
		return false
	}
	return browserauth.VerifyCSRF(
		refreshToken,
		browserauth.CSRFToken(c.Request),
		c.GetHeader(browserauth.CSRFHeaderName),
		secret,
	)
}

func requestHTTPS(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

// EstablishBrowserSession migrates a token pair returned by an older login
// path into a cookie session. Both the authenticated access token and the
// matching refresh token are required; an access token alone cannot mint a
// long-lived session.
func (h *AuthHandler) EstablishBrowserSession(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || !BrowserSessionRequested(c) {
		response.Forbidden(c, "Browser session is required")
		return
	}
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "A refresh token is required")
		return
	}
	result, err := h.authService.RefreshTokenPair(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	claims, err := h.authService.ValidateToken(result.AccessToken)
	if err != nil || claims.UserID != subject.UserID {
		_ = h.authService.RevokeRefreshToken(c.Request.Context(), result.RefreshToken)
		response.Forbidden(c, "Session user mismatch")
		return
	}
	if err := SetBrowserSessionCookies(c, result.RefreshToken, h.cfg.JWT.Secret, h.cfg.JWT.RefreshTokenExpireDays*24*60*60); err != nil {
		response.InternalError(c, "Failed to establish browser session")
		return
	}
	response.Success(c, RefreshTokenResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		TokenType:   "Bearer",
	})
}
