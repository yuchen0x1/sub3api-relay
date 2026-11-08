//go:build unit

package web

import (
	"mime"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebManifestMimeTypeRegistered(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "application/manifest+json", mime.TypeByExtension(".webmanifest"))
}

func TestApplyPWACacheHeaders(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"sw.js", "manifest.webmanifest", "/sw.js"} {
		path := path
		t.Run("revalidates_"+path, func(t *testing.T) {
			t.Parallel()
			header := make(http.Header)
			applyPWACacheHeaders(header, path)
			assert.Equal(t, "no-cache", header.Get("Cache-Control"))
		})
	}

	for _, path := range []string{"index.html", "assets/index-AbCd1234.js", "logo.svg"} {
		path := path
		t.Run("skips_"+path, func(t *testing.T) {
			t.Parallel()
			header := make(http.Header)
			applyPWACacheHeaders(header, path)
			assert.Empty(t, header.Get("Cache-Control"))
		})
	}

	t.Run("nil_header_is_noop", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			applyPWACacheHeaders(nil, "sw.js")
		})
	})
}
