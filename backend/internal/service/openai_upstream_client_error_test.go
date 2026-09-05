package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func TestHandleErrorResponse_Deterministic400IsNotRewrappedAs502(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := `{"error":{"message":"Invalid schema","type":"invalid_request_error","param":"input[0]","code":"invalid_function_parameters"}}`
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Name: "acct"}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.Get(rec.Body.String(), "error.type").String())
	require.Equal(t, "invalid_function_parameters", gjson.Get(rec.Body.String(), "error.code").String())
	require.Equal(t, "input[0]", gjson.Get(rec.Body.String(), "error.param").String())
	require.Equal(t, "Invalid schema", gjson.Get(rec.Body.String(), "error.message").String())
}

func TestWriteOpenAIUpstreamClientError_UsesSanitizedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeOpenAIUpstreamClientError(c, http.StatusBadRequest, []byte(`{"error":{"message":"key=secret123"}}`), "key=***")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "key=***", gjson.Get(rec.Body.String(), "error.message").String())
	require.NotContains(t, rec.Body.String(), "secret123")
}
