package handler

import (
	"fmt"
	"strings"
	"testing"

	coderws "github.com/coder/websocket"
)

func TestSummarizeWSCloseErrorForLogOmitsPeerPayload(t *testing.T) {
	err := fmt.Errorf("upstream close: %w", coderws.CloseError{
		Code: coderws.StatusPolicyViolation, Reason: "echoed-proxy-password bearer-token",
	})
	status, reason := summarizeWSCloseErrorForLog(err)
	if !strings.Contains(status, "1008") {
		t.Fatalf("lost close status: %q", status)
	}
	if strings.Contains(reason, "echoed-proxy-password") || strings.Contains(reason, "bearer-token") {
		t.Fatalf("logged peer close payload: %q", reason)
	}
	if !strings.Contains(reason, "type=websocket.CloseError") {
		t.Fatalf("lost diagnostic error type: %q", reason)
	}
}
