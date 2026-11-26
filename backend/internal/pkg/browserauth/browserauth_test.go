package browserauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCookiesAndCSRF(t *testing.T) {
	const refresh = "refresh-token"
	const secret = "test-secret"
	w := httptest.NewRecorder()
	require.NoError(t, SetCookies(w, refresh, secret, 3600, true, http.SameSiteLaxMode))
	result := w.Result()
	cookies := result.Cookies()
	require.Len(t, cookies, 3)
	var refreshCookie, csrfCookie, sessionCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case RefreshCookieName:
			refreshCookie = cookie
		case CSRFCookieName:
			csrfCookie = cookie
		case SessionCookieName:
			sessionCookie = cookie
		}
	}
	require.NotNil(t, refreshCookie)
	require.NotNil(t, csrfCookie)
	require.NotNil(t, sessionCookie)
	require.True(t, refreshCookie.HttpOnly)
	require.False(t, csrfCookie.HttpOnly)
	require.True(t, refreshCookie.Secure)
	require.Equal(t, CookiePath, refreshCookie.Path)
	require.Equal(t, CSRFCookiePath, csrfCookie.Path, "the SPA must be able to read the CSRF cookie")
	require.True(t, VerifyCSRF(refresh, csrfCookie.Value, csrfCookie.Value, secret))
	require.False(t, VerifyCSRF(refresh, csrfCookie.Value, csrfCookie.Value, "wrong"))
	require.False(t, VerifyCSRF(refresh+"-other", csrfCookie.Value, csrfCookie.Value, secret))
	require.False(t, VerifyCSRF(refresh, csrfCookie.Value, "attacker", secret))
}

func TestClearCookiesUsesMatchingPaths(t *testing.T) {
	w := httptest.NewRecorder()
	ClearCookies(w, true, http.SameSiteLaxMode)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 3)
	for _, cookie := range cookies {
		require.Equal(t, -1, cookie.MaxAge)
		if cookie.Name == RefreshCookieName {
			require.Equal(t, CookiePath, cookie.Path)
			require.True(t, cookie.HttpOnly)
		} else {
			require.Equal(t, CSRFCookiePath, cookie.Path)
		}
	}
}

func TestRequestedRequiresExplicitMode(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	require.False(t, Requested(r))
	r.Header.Set(SessionModeHeader, "COOKIE")
	require.True(t, Requested(r))
	r.Header.Set(SessionModeHeader, "bearer")
	require.False(t, Requested(r))
}
