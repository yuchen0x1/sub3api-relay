package service

import (
	"mime"
	"net/http"
	"strings"
)

// setOpenAIJSONResponseType preserves the response bytes while preventing a
// mislabeled upstream response from being interpreted as an active document.
func setOpenAIJSONResponseType(header http.Header, contentType string) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || (mediaType != "application/json" && (!strings.HasPrefix(mediaType, "application/") || !strings.HasSuffix(mediaType, "+json"))) {
		contentType = "application/json"
	}
	header.Set("Content-Type", contentType)
	header.Set("X-Content-Type-Options", "nosniff")
}
