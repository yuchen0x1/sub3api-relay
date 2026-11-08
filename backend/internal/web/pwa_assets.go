//go:build embed || unit

package web

import (
	"mime"
	"net/http"
	"strings"
)

const (
	// serviceWorkerFile is the generated Workbox service worker.
	serviceWorkerFile = "sw.js"
	// webManifestFile is the generated PWA manifest.
	webManifestFile = "manifest.webmanifest"
)

// Go's mime table has no entry for .webmanifest, so http.FileServer would
// sniff the payload and hand Chrome an unknown content type. Chrome only
// accepts a manifest served as application/manifest+json.
func init() {
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		panic("register webmanifest mime type: " + err.Error())
	}
}

// applyPWACacheHeaders keeps the service worker and manifest revalidated on
// every request. The worker script is byte-compared by the browser to detect
// updates, so a long-lived cache would delay new releases.
func applyPWACacheHeaders(header http.Header, cleanPath string) {
	if header == nil {
		return
	}

	switch strings.TrimPrefix(cleanPath, "/") {
	case serviceWorkerFile, webManifestFile:
		header.Set("Cache-Control", "no-cache")
	}
}
