package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestProxyExportDataPreservesHysteria2TLSSetting(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()
	adminSvc.proxies = []service.Proxy{{
		ID:          1,
		Name:        "hy2",
		Protocol:    "hysteria2",
		Host:        "hy.example.com",
		Port:        443,
		Password:    "secret",
		TLSInsecure: true,
		Status:      service.StatusActive,
	}}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/data", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var result proxyDataResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Len(t, result.Data.Proxies, 1)
	require.True(t, result.Data.Proxies[0].TLSInsecure)
	require.Contains(t, result.Data.Proxies[0].ProxyKey, "tls-insecure")
}

func TestProxyImportDataPreservesHysteria2TLSSetting(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()
	payload := map[string]any{"data": map[string]any{
		"type": dataType, "version": dataVersion,
		"proxies": []map[string]any{{
			"name": "hy2", "protocol": "hysteria2", "host": "hy.example.com", "port": 443,
			"password": "secret", "tls_insecure": true, "status": "active",
		}},
		"accounts": []map[string]any{},
	}}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/data", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, adminSvc.createdProxies, 1)
	require.Equal(t, "hysteria2", adminSvc.createdProxies[0].Protocol)
	require.Equal(t, "secret", adminSvc.createdProxies[0].Password)
	require.True(t, adminSvc.createdProxies[0].TLSInsecure)
}
