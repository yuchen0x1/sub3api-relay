package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type resetFingerprintCacheStub struct {
	deleted []int64
}

func (s *resetFingerprintCacheStub) GetFingerprint(_ context.Context, _ int64) (*service.Fingerprint, error) {
	return nil, nil
}

func (s *resetFingerprintCacheStub) SetFingerprint(_ context.Context, _ int64, _ *service.Fingerprint) error {
	return nil
}

func (s *resetFingerprintCacheStub) DeleteFingerprint(_ context.Context, accountID int64) error {
	s.deleted = append(s.deleted, accountID)
	return nil
}

func (s *resetFingerprintCacheStub) GetMaskedSessionID(_ context.Context, _ int64) (string, error) {
	return "", nil
}

func (s *resetFingerprintCacheStub) SetMaskedSessionID(_ context.Context, _ int64, _ string) error {
	return nil
}

func setupResetFingerprintRouter(cache *resetFingerprintCacheStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageService := service.NewAccountUsageService(nil, nil, nil, nil, nil, nil, nil, nil, nil, cache, nil)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/admin/accounts/:id/reset-fingerprint", handler.ResetFingerprint)
	return router
}

func TestAccountHandlerResetFingerprintDeletesCachedFingerprint(t *testing.T) {
	cache := &resetFingerprintCacheStub{}
	router := setupResetFingerprintRouter(cache)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/admin/accounts/42/reset-fingerprint", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []int64{42}, cache.deleted)

	var response struct {
		Data struct {
			Reset bool `json:"reset"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Data.Reset)
}

func TestAccountHandlerResetFingerprintRejectsInvalidID(t *testing.T) {
	cache := &resetFingerprintCacheStub{}
	router := setupResetFingerprintRouter(cache)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/admin/accounts/nope/reset-fingerprint", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, cache.deleted)
}
