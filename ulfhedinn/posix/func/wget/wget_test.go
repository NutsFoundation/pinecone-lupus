package wget

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

func TestWgetDownloadsToOutputFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	dir := t.TempDir()
	var stderr bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:    []string{"-q", "-O", "payload.bin", server.URL},
		WorkDir: dir,
		Stderr:  &stderr,
	})

	if result.ExitCode != cli.ExitSuccess {
		t.Fatalf("wget failed: %#v stderr=%s", result, stderr.String())
	}
	content, err := os.ReadFile(filepath.Join(dir, "payload.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "payload" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestWgetFailsOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:   []string{"-O", filepath.Join(t.TempDir(), "payload.bin"), server.URL},
		Stderr: &stderr,
	})

	if result.ExitCode != cli.ExitHTTPFailure {
		t.Fatalf("unexpected result: %#v stderr=%s", result, stderr.String())
	}
}

func TestWgetWritesToStdoutWithDashOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:   []string{"-q", "-O", "-", server.URL},
		Stdout: &stdout,
	})

	if result.ExitCode != cli.ExitSuccess {
		t.Fatalf("wget failed: %#v", result)
	}
	if stdout.String() != "payload" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}
