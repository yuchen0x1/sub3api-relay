//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestIsGrokCompactionReplayDecodeError(t *testing.T) {
	matching := [][]byte{
		[]byte(`{"error":{"message":"failed to decode compaction item in response history"}}`),
		[]byte(`{"error":{"message":"unable to deserialize summary from encrypted_content"}}`),
		[]byte(`plain text: could not decode response history blob`),
	}
	for _, body := range matching {
		require.True(t, isGrokCompactionReplayDecodeError(http.StatusBadRequest, body))
		require.True(t, isGrokCompactionReplayDecodeError(http.StatusUnprocessableEntity, body))
	}

	notMatching := []struct {
		status int
		body   []byte
	}{
		{http.StatusInternalServerError, []byte(`{"error":{"message":"failed to decode compaction item"}}`)},
		{http.StatusBadRequest, []byte(`{"error":{"message":"invalid model"}}`)},
		{http.StatusBadRequest, []byte(`{"error":{"message":"decrypt failed"}}`)},
		{http.StatusBadRequest, nil},
	}
	for _, tc := range notMatching {
		require.False(t, isGrokCompactionReplayDecodeError(tc.status, tc.body))
	}
}

func TestSanitizeGrokCompactionReplayBodyDropsEmptyReasoning(t *testing.T) {
	body := []byte(`{"input":[
		{"type":"reasoning"},
		{"type":"message","role":"user","content":"hi"},
		{"type":"reasoning","summary":[]},
		{"type":"reasoning","summary":[{"type":"summary_text","summary":"kept visible summary"}],"encrypted_content":"enc"}
	]}`)

	retryBody, changed, err := sanitizeGrokCompactionReplayBody(body)
	require.NoError(t, err)
	require.True(t, changed)

	items := gjson.GetBytes(retryBody, "input").Array()
	require.Len(t, items, 2, "empty reasoning and opaque encrypted state dropped; message and visible summary kept")
	require.Equal(t, "message", items[0].Get("type").String())
	require.Equal(t, "reasoning", items[1].Get("type").String())
	require.Equal(t, "kept visible summary", items[1].Get("summary.0.summary").String())
}

func TestSanitizeGrokCompactionReplayBodyNoChanges(t *testing.T) {
	body := []byte(`{"input":[{"type":"message","role":"user","content":"hi"}]}`)
	retryBody, changed, err := sanitizeGrokCompactionReplayBody(body)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, body, retryBody)
}
