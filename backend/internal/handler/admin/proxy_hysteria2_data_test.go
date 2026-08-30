package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyImportDataRejectsHysteria2(t *testing.T) {
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
	require.Empty(t, adminSvc.createdProxies)

	var result proxyImportResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, 1, result.Data.ProxyFailed)
	require.Contains(t, result.Data.Errors[0].Message, "proxy protocol is invalid")
}
