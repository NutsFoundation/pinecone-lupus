package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

func TestDownloaderWritesHTTPResponseAtomically(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "payload.bin")
	result := NewDownloader(DownloadConfig{}).Download(context.Background(), DownloadRequest{
		URL:        server.URL,
		OutputPath: target,
	})

	if !result.Success || result.ExitCode != cli.ExitSuccess {
		t.Fatalf("download failed: %#v", result)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "payload" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestDownloaderFailsOnHTTPErrorWhenRequested(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte("missing"))
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "payload.bin")
	result := NewDownloader(DownloadConfig{}).Download(context.Background(), DownloadRequest{
		URL:             server.URL,
		OutputPath:      target,
		FailOnHTTPError: true,
	})

	if result.Success {
		t.Fatalf("expected failure: %#v", result)
	}
	if result.ExitCode != cli.ExitHTTPFailure {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target should not exist after failed atomic write, stat err=%v", err)
	}
}

func TestDownloaderFollowsRedirects(t *testing.T) {
	server := httptest.NewServer(nil)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			w.Header().Set("Location", "/payload")
			w.WriteHeader(302)
			return
		}
		_, _ = w.Write([]byte("redirected"))
	})
	defer server.Close()

	target := filepath.Join(t.TempDir(), "payload.bin")
	result := NewDownloader(DownloadConfig{}).Download(context.Background(), DownloadRequest{
		URL:             server.URL + "/redirect",
		OutputPath:      target,
		FollowRedirects: true,
	})

	if !result.Success {
		t.Fatalf("download failed: %#v", result)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "redirected" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestDownloaderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("slow"))
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "payload.bin")
	result := NewDownloader(DownloadConfig{}).Download(context.Background(), DownloadRequest{
		URL:           server.URL,
		OutputPath:    target,
		TimeoutMillis: 1,
	})

	if result.Success {
		t.Fatalf("expected timeout failure: %#v", result)
	}
	if result.ExitCode != cli.ExitTimeout {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}
}
