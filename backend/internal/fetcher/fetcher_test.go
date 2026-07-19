package fetcher_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/emailservice/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFetcher() fetcher.Fetcher {
	return fetcher.NewCombinedFetcher()
}

func TestFetchHTTP_Success(t *testing.T) {
	content := []byte("pdf content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(content)
		if err != nil {
			panic(err)
		}
	}))
	defer srv.Close()

	f := newFetcher()
	tmpPath, cleanup, err := f.Fetch(srv.URL+"/file.pdf", "attachment.pdf")
	require.NoError(t, err)
	require.NotEmpty(t, tmpPath)
	defer cleanup()

	data, err := os.ReadFile(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestFetchHTTP_Non200_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	f := newFetcher()
	_, cleanup, err := f.Fetch(srv.URL+"/secret.pdf", "secret.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	if cleanup != nil {
		cleanup()
	}
}

func TestFetchFile_LocalFile_Success(t *testing.T) {
	// Write a temp file on disk
	dir := t.TempDir()
	fPath := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(fPath, content, 0644))

	f := newFetcher()
	tmpPath, cleanup, err := f.Fetch("file://"+fPath, "test.txt")
	require.NoError(t, err)
	defer cleanup()

	data, err := os.ReadFile(tmpPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestFetchFile_Missing_ReturnsError(t *testing.T) {
	f := newFetcher()
	_, cleanup, err := f.Fetch("file:///nonexistent/path/file.pdf", "file.pdf")
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestFetch_EmptyURL_ReturnsError(t *testing.T) {
	f := newFetcher()
	_, cleanup, err := f.Fetch("", "test.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty attachment url")
	if cleanup != nil {
		cleanup()
	}
}

func TestFetch_UnsupportedScheme_ReturnsError(t *testing.T) {
	f := newFetcher()
	_, cleanup, err := f.Fetch("ftp://some.host/file.pdf", "file.pdf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scheme")
	if cleanup != nil {
		cleanup()
	}
}

func TestFetch_Cleanup_RemovesTempFile(t *testing.T) {
	content := []byte("data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(content)
		if err != nil {
			panic(err)
		}
	}))
	defer srv.Close()

	f := newFetcher()
	tmpPath, cleanup, err := f.Fetch(srv.URL+"/data.bin", "data.bin")
	require.NoError(t, err)

	// Verify temp file exists before cleanup
	_, statErr := os.Stat(tmpPath)
	assert.NoError(t, statErr)

	// Cleanup should remove the temp file
	cleanup()
	_, statErr = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(statErr), "temp file should be removed after cleanup")
}
