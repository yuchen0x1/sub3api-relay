package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/ctxkey"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// GatewayBodyLog 记录网关请求体（设置中心开关，默认关闭）。
//
// 实现：预读请求体（受网关 MaxBodySize 同额限制）后把 Body 换成可重读 reader，
// 下游 handler 行为不变；功能关闭时零开销直通。落库为 fire-and-forget。
// 超限请求走 MultiReader 还原原始流，保证下游 MaxBytesReader 的 413 语义不变。
func GatewayBodyLog(recorder *service.GatewayBodyLogService, maxBodyBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recorder == nil || !recorder.IsEnabled(c.Request.Context()) ||
			c.Request.Method != http.MethodPost || c.Request.Body == nil {
			c.Next()
			return
		}
		if maxBodyBytes <= 0 {
			maxBodyBytes = service.MaxGatewayBodyLogBytes
		}

		limited := io.LimitReader(c.Request.Body, maxBodyBytes+1)
		data, readErr := io.ReadAll(limited)
		if readErr != nil {
			// 读取中途失败：把已读内容拼回原流，让下游按原样处理并报错
			c.Request.Body = readCloser{
				Reader: io.MultiReader(bytes.NewReader(data), c.Request.Body),
				Closer: c.Request.Body,
			}
			c.Next()
			return
		}
		if int64(len(data)) > maxBodyBytes {
			// 超过网关上限：还原完整流，交由下游 MaxBytesReader 正常拒绝
			c.Request.Body = readCloser{
				Reader: io.MultiReader(bytes.NewReader(data), c.Request.Body),
				Closer: c.Request.Body,
			}
			c.Next()
			return
		}
		_ = c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(data))

		entry := &service.GatewayBodyLogEntry{
			RequestID: requestIDFromContext(c),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Model:     gjson.GetBytes(data, "model").String(),
		}
		if apiKey, ok := GetAPIKeyFromContext(c); ok && apiKey != nil {
			entry.APIKeyID = apiKey.ID
		}
		if subject, ok := GetAuthSubjectFromContext(c); ok {
			entry.UserID = subject.UserID
		}
		entry.Body = string(data)
		recorder.Record(c.Request.Context(), entry)

		c.Next()
	}
}

// requestIDFromContext 取请求关联 ID（RequestLogger 中间件已生成）。
func requestIDFromContext(c *gin.Context) string {
	if v, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
		return v
	}
	return ""
}

// readCloser 组合 Reader 与 Closer，用于还原请求体流。
type readCloser struct {
	io.Reader
	io.Closer
}
