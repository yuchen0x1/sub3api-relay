package browserauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const (
	SessionModeHeader = "X-Sub3API-Browser-Session"
	SessionModeCookie = "cookie"
	RefreshCookieName = "sub3api_browser_refresh"
	CSRFCookieName    = "sub3api_browser_csrf"
	SessionCookieName = "sub3api_browser_session"
	CSRFHeaderName    = "X-CSRF-Token"
	CookiePath        = "/api/v1/auth"
	CSRFCookiePath    = "/"
)

var ErrInvalidCSRF = errors.New("invalid browser csrf token")

// Requested reports whether the caller explicitly opted into browser-cookie mode.
// Legacy bearer clients never send this header and therefore keep their existing contract.
func Requested(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get(SessionModeHeader)), SessionModeCookie)
}

func RefreshToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(RefreshCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func CSRFToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// NewCSRFToken creates a signed double-submit token bound to the refresh token.
// The random nonce prevents token reuse across sessions; the HMAC prevents a hostile
// subdomain from replacing the readable CSRF cookie with an arbitrary value.
func NewCSRFToken(refreshToken, secret string) (string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	secret = strings.TrimSpace(secret)
	if refreshToken == "" || secret == "" {
		return "", ErrInvalidCSRF
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	nonceValue := base64.RawURLEncoding.EncodeToString(nonce)
	return nonceValue + "." + signCSRF(refreshToken, nonceValue, secret), nil
}

func VerifyCSRF(refreshToken, cookieValue, headerValue, secret string) bool {
	refreshToken = strings.TrimSpace(refreshToken)
	cookieValue = strings.TrimSpace(cookieValue)
	headerValue = strings.TrimSpace(headerValue)
	secret = strings.TrimSpace(secret)
	if refreshToken == "" || cookieValue == "" || headerValue == "" || secret == "" || cookieValue != headerValue {
		return false
	}
	nonce, signature, ok := strings.Cut(cookieValue, ".")
	if !ok || nonce == "" || signature == "" {
		return false
	}
	if _, err := base64.RawURLEncoding.DecodeString(nonce); err != nil {
		return false
	}
	expected := signCSRF(refreshToken, nonce, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

func SetCookies(w http.ResponseWriter, refreshToken, secret string, maxAge int, secure bool, sameSite http.SameSite) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ErrInvalidCSRF
	}
	csrfToken, err := NewCSRFToken(refreshToken, secret)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    refreshToken,
		Path:     CookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken,
		Path:     CSRFCookiePath,
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   secure,
		SameSite: sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "1",
		Path:     CSRFCookiePath,
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   secure,
		SameSite: sameSite,
	})
	return nil
}

func ClearCookies(w http.ResponseWriter, secure bool, sameSite http.SameSite) {
	for _, name := range []string{RefreshCookieName, CSRFCookieName, SessionCookieName} {
		path := CookiePath
		if name != RefreshCookieName {
			path = CSRFCookiePath
		}
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     path,
			MaxAge:   -1,
			HttpOnly: name == RefreshCookieName,
			Secure:   secure,
			SameSite: sameSite,
		})
	}
}

func signCSRF(refreshToken, nonce, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("sub3api-browser-csrf\x00"))
	_, _ = mac.Write([]byte(refreshToken))
	_, _ = mac.Write([]byte("\x00"))
	_, _ = mac.Write([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
