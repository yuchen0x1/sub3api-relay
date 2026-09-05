package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateServiceExtractBinaryFromZip(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "sub3api_windows_amd64.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)

	zw := zip.NewWriter(file)
	entry, err := zw.Create("sub3api.exe")
	require.NoError(t, err)
	_, err = entry.Write([]byte("sub3api-binary"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, file.Close())

	destPath := filepath.Join(t.TempDir(), "server.exe")
	require.NoError(t, (&UpdateService{}).extractBinary(archivePath, destPath))

	contents, err := os.ReadFile(destPath)
	require.NoError(t, err)
	require.Equal(t, []byte("sub3api-binary"), contents)
}

func TestUpdateServiceExtractBinaryFromZipRejectsTraversal(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "sub3api_windows_amd64.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)

	zw := zip.NewWriter(file)
	entry, err := zw.Create("../sub3api.exe")
	require.NoError(t, err)
	_, err = entry.Write([]byte("sub3api-binary"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, file.Close())

	err = (&UpdateService{}).extractBinary(archivePath, filepath.Join(t.TempDir(), "server.exe"))
	require.ErrorContains(t, err, "path traversal attempt detected")
}
