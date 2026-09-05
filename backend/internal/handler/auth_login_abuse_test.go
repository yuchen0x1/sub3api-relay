package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type authLoginAbuseUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *authLoginAbuseUserRepo) GetByEmail(_ context.Context, email string) (*service.User, error) {
	if r.user != nil && strings.EqualFold(strings.TrimSpace(email), r.user.Email) {
		return r.user, nil
	}
	return nil, service.ErrUserNotFound
}

func TestAuthLoginTemporarilyLocksAfterRepeatedInvalidPasswords(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &service.User{
		ID:     1,
		Email:  "admin@sub2api.local",
		Role:   service.RoleAdmin,
		Status: service.StatusActive,
	}
	require.NoError(t, user.SetPassword("correct-password"))

	authSvc := service.NewAuthService(
		nil,
		&authLoginAbuseUserRepo{user: user},
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	h := &AuthHandler{authService: authSvc}

	router := gin.New()
	router.POST("/api/v1/auth/login", h.Login)

	for i := 1; i < 5; i++ {
		w := performAuthLoginAbuseRequest(router, "{\"email\":\"admin@sub2api.local\",\"password\":\"wrong\"}")
		require.Equal(t, http.StatusUnauthorized, w.Code, "attempt %d", i)
	}

	locked := performAuthLoginAbuseRequest(router, "{\"email\":\"admin@sub2api.local\",\"password\":\"wrong\"}")
	require.Equal(t, http.StatusTooManyRequests, locked.Code)
	require.Equal(t, "1800", locked.Header().Get("Retry-After"))
	require.Contains(t, locked.Body.String(), "AUTH_LOGIN_TEMPORARILY_LOCKED")

	stillLocked := performAuthLoginAbuseRequest(router, "{\"email\":\"admin@sub2api.local\",\"password\":\"correct-password\"}")
	require.Equal(t, http.StatusTooManyRequests, stillLocked.Code)
	require.Contains(t, stillLocked.Body.String(), "AUTH_LOGIN_TEMPORARILY_LOCKED")
}

func performAuthLoginAbuseRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "13.158.34.225:53210"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
