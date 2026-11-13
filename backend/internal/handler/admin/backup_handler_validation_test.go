package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateBackupRejectsMalformedRequest(t *testing.T) {
	for _, body := range []string{`{"expire_days":"invalid"}`, `{"expire_days":`, `{"expire_days":1.5}`} {
		t.Run(body, func(t *testing.T) {
			router := gin.New()
			// A malformed request must be rejected before touching the service.
			router.POST("/backups", (&BackupHandler{}).CreateBackup)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/backups", strings.NewReader(body)))
			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
