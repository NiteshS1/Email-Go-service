package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type CombinedFetcher struct{}

func NewCombinedFetcher() Fetcher {
	return &CombinedFetcher{}
}

func (f *CombinedFetcher) Fetch(resourceURL string, name string) (tempPath string, cleanup func(), err error) {
	if resourceURL == "" {
		return "", nil, fmt.Errorf("empty attachment url")
	}

	u, err := url.Parse(resourceURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid url: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme == "" && (strings.HasPrefix(resourceURL, "/") || filepath.VolumeName(resourceURL) != "") {
		u, _ = url.Parse("file://" + resourceURL)
		scheme = "file"
	}

	switch scheme {
	case "file":
		return f.fetchFile(u, name)
	case "http", "https":
		return f.fetchHTTP(resourceURL, name)
	default:
		return "", nil, fmt.Errorf("unsupported scheme %q (use file://, http://, https://, or a path like /path/to/file; s3 later)", u.Scheme)
	}
}

func (f *CombinedFetcher) fetchFile(u *url.URL, name string) (tempPath string, cleanup func(), err error) {
	path := u.Path
	if u.Host != "" && u.Host != "localhost" {
		path = filepath.Join(u.Host, u.Path)
	}
	src, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	ext := filepath.Ext(name)
	if ext == "" {
		ext = filepath.Ext(path)
	}
	tmp, err := os.CreateTemp("", "email-attachment-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	_, err = io.Copy(tmp, src)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", nil, fmt.Errorf("copy to temp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", nil, err
	}
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func (f *CombinedFetcher) fetchHTTP(resourceURL string, name string) (tempPath string, cleanup func(), err error) {
	resp, err := http.Get(resourceURL)
	if err != nil {
		return "", nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".bin"
	}
	tmp, err := os.CreateTemp("", "email-attachment-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	_, err = io.Copy(tmp, resp.Body)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", nil, fmt.Errorf("download to temp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", nil, err
	}
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}
