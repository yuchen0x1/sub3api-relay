package repository

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/config"
)

func TestBackupRestoreCommandStopsOnErrors(t *testing.T) {
	dumper := &PgDumper{cfg: &config.DatabaseConfig{Host: "localhost", Port: 5432, User: "test", DBName: "test", Password: "fixture-only", SSLMode: "require"}}
	cmd := dumper.restoreCommand(context.Background())
	// psql only applies --single-transaction when -c or -f is supplied.
	for _, arg := range []string{"--file=-", "--single-transaction", "--set=ON_ERROR_STOP=on", "--no-psqlrc"} {
		require.Contains(t, cmd.Args, arg)
	}
	require.NotContains(t, strings.Join(cmd.Args, " "), "fixture-only")
	require.Contains(t, cmd.Env, "PGSSLMODE=require")
}

type backupReadFailure struct{}

func (backupReadFailure) Read([]byte) (int, error) { return 0, errors.New("dump interrupted") }

func TestBackupUploadSpool(t *testing.T) {
	for _, mode := range []string{"success", "read-error", "upload-error"} {
		t.Run(mode, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("TMP", tmp)
			t.Setenv("TEMP", tmp)
			payload := strings.Repeat("backup bytes", 10000)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != payload || r.ContentLength != int64(len(payload)) {
					t.Error("upload payload or content length mismatch")
				}
				if mode == "upload-error" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", "")}, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(server.URL)
				o.UsePathStyle = true
				o.RetryMaxAttempts = 1
			})
			store := &S3BackupStore{client: client, bucket: "test"}
			var reader io.Reader = strings.NewReader(payload)
			if mode == "read-error" {
				reader = backupReadFailure{}
			}
			size, err := store.Upload(context.Background(), "backup.gz", reader, "application/gzip")
			if mode == "success" {
				require.NoError(t, err)
				require.Equal(t, int64(len(payload)), size)
			} else {
				require.Error(t, err)
			}
			if mode == "read-error" {
				require.Zero(t, requests)
			}
			files, err := os.ReadDir(tmp)
			require.NoError(t, err)
			require.Empty(t, files, "temporary backup must be removed on success and failure")
		})
	}
}
