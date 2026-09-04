package repository

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// newAliyunCaptchaTestTarget 起一个假的阿里云端点，让真实 SDK 走完整的签名/序列化链路。
func newAliyunCaptchaTestTarget(t *testing.T, handler http.HandlerFunc) (*aliyunCaptchaVerifier, service.AliyunCaptchaCredentials) {
	t.Helper()
	// tea SDK 会继承宿主代理环境变量，且无法解析不带 scheme 的代理地址
	// （如本机 Clash 常见的 http_proxy=127.0.0.1:7897），导致与被测逻辑无关的失败。
	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY", "all_proxy", "ALL_PROXY"} {
		t.Setenv(key, "")
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	verifier := &aliyunCaptchaVerifier{protocol: "HTTP", timeoutMillis: 2_000}
	cred := service.AliyunCaptchaCredentials{
		AccessKeyID:     "test-ak-id",
		AccessKeySecret: "test-ak-secret",
		SceneID:         "scene-1",
		Endpoint:        strings.TrimPrefix(server.URL, "http://"),
	}
	return verifier, cred
}

func TestAliyunCaptchaVerifier_VerifySuccess(t *testing.T) {
	var capturedParam, capturedSceneID string
	verifier, cred := newAliyunCaptchaTestTarget(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		capturedParam = r.Form.Get("CaptchaVerifyParam")
		capturedSceneID = r.Form.Get("SceneId")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":"Success","Message":"success","RequestId":"req-1","Success":true,"Result":{"VerifyResult":true,"VerifyCode":"T001"}}`))
	})

	result, err := verifier.VerifyCaptcha(context.Background(), cred, "the-verify-param")
	require.NoError(t, err)
	require.True(t, result.VerifyResult)
	require.Equal(t, "T001", result.VerifyCode)
	require.Equal(t, "the-verify-param", capturedParam)
	require.Equal(t, "scene-1", capturedSceneID)
}

func TestAliyunCaptchaVerifier_VerifyResultFalse(t *testing.T) {
	verifier, cred := newAliyunCaptchaTestTarget(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":"Success","RequestId":"req-2","Success":true,"Result":{"VerifyResult":false,"VerifyCode":"F002"}}`))
	})

	result, err := verifier.VerifyCaptcha(context.Background(), cred, "bad-param")
	require.NoError(t, err)
	require.False(t, result.VerifyResult)
	require.Equal(t, "F002", result.VerifyCode)
}

func TestAliyunCaptchaVerifier_APIErrorNormalized(t *testing.T) {
	verifier, cred := newAliyunCaptchaTestTarget(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"Code":"SignatureDoesNotMatch","Message":"Specified signature is not matched with our calculation.","RequestId":"req-3"}`))
	})

	_, err := verifier.VerifyCaptcha(context.Background(), cred, "param")
	require.Error(t, err)
	var apiErr *service.AliyunCaptchaAPIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, "SignatureDoesNotMatch", apiErr.Code)
}

func TestAliyunCaptchaVerifier_TransportError(t *testing.T) {
	for _, key := range []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY", "all_proxy", "ALL_PROXY"} {
		t.Setenv(key, "")
	}
	server := httptest.NewServer(http.NotFoundHandler())
	endpoint := strings.TrimPrefix(server.URL, "http://")
	server.Close() // 立即关闭，制造连接失败

	verifier := &aliyunCaptchaVerifier{protocol: "HTTP", timeoutMillis: 2_000}
	cred := service.AliyunCaptchaCredentials{
		AccessKeyID:     "test-ak-id",
		AccessKeySecret: "test-ak-secret",
		SceneID:         "scene-1",
		Endpoint:        endpoint,
	}

	_, err := verifier.VerifyCaptcha(context.Background(), cred, "param")
	require.Error(t, err)
	var apiErr *service.AliyunCaptchaAPIError
	require.False(t, errors.As(err, &apiErr), "transport errors must not be normalized to API errors")
}
