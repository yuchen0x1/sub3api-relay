package handler

import (
	"strconv"
	"strings"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/ctxkey"
	pkghttputil "github.com/yuchen0x1/sub3api-relay/internal/pkg/httputil"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// parseFailureSnippetLen bounds the head/tail snippets logged on body parse
// failure. 256 bytes is enough to see the structural context (model field,
// first content block / trailing brace) without dumping user payloads.
const parseFailureSnippetLen = 256

// logRequestBodyReadFailure records transfer metadata without logging payload
// bytes, allowing operators to distinguish client disconnects from decoder and
// server-side read failures.
func logRequestBodyReadFailure(c *gin.Context, reqLog *zap.Logger, err error) {
	if reqLog == nil || err == nil {
		return
	}

	fields := []zap.Field{
		zap.Error(err),
		zap.Bool("client_disconnected", pkghttputil.IsRequestBodyClientDisconnect(err)),
	}
	if bytesRead, contentLength, ok := pkghttputil.RequestBodyReadProgress(err); ok {
		fields = append(fields,
			zap.Int64("bytes_read", bytesRead),
			zap.Int64("content_length", contentLength),
		)
	}
	if c != nil && c.Request != nil {
		if encoding := strings.TrimSpace(c.Request.Header.Get("Content-Encoding")); encoding != "" {
			fields = append(fields, zap.String("content_encoding", encoding))
		}
		if len(c.Request.TransferEncoding) > 0 {
			fields = append(fields, zap.Strings("transfer_encoding", c.Request.TransferEncoding))
		}
		if ctxErr := c.Request.Context().Err(); ctxErr != nil {
			fields = append(fields, zap.String("request_context_error", ctxErr.Error()))
		}
		if clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); clientRequestID != "" {
			fields = append(fields, zap.String("client_request_id", clientRequestID))
		}
	}

	reqLog.Warn("read request body failed", fields...)
}

// logRequestBodyParseFailure records the real reason a request body failed
// JSON parsing/validation. The client keeps receiving the generic
// "Failed to parse request body"; the sanitized diagnostics (underlying
// error with byte offset, body length, escaped head/tail snippets) land in
// the server log only, so operators can distinguish genuinely invalid JSON
// from a truncated or partially consumed body.
//
// err may be nil for call sites that validate with gjson.ValidBytes directly;
// the diagnostic error is derived from the body in that case.
func logRequestBodyParseFailure(reqLog *zap.Logger, body []byte, err error) {
	if reqLog == nil {
		return
	}
	if err == nil {
		err = service.DescribeInvalidJSON(body)
	}

	head := body
	var tail []byte
	if len(body) > parseFailureSnippetLen {
		head = body[:parseFailureSnippetLen]
		tail = body[len(body)-parseFailureSnippetLen:]
	}

	fields := []zap.Field{
		zap.Error(err),
		zap.Int("body_len", len(body)),
		zap.String("body_head", sanitizeBodySnippet(head)),
	}
	if len(tail) > 0 {
		fields = append(fields, zap.String("body_tail", sanitizeBodySnippet(tail)))
	}
	reqLog.Warn("parse request body failed", fields...)
}

// sanitizeBodySnippet escapes control characters and invalid UTF-8 so the
// snippet is always a single printable log line.
func sanitizeBodySnippet(b []byte) string {
	return strconv.Quote(string(b))
}
