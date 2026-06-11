package curl

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

func TestCurlDownloadsWithCombinedFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	dir := t.TempDir()
	var stderr bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:    []string{"-fsSL", "-o", "payload.bin", server.URL},
		WorkDir: dir,
		Stderr:  &stderr,
	})

	if result.ExitCode != cli.ExitSuccess {
		t.Fatalf("curl failed: %#v stderr=%s", result, stderr.String())
	}
	content, err := os.ReadFile(filepath.Join(dir, "payload.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "payload" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestCurlWritesToStdoutByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:   []string{server.URL},
		Stdout: &stdout,
	})

	if result.ExitCode != cli.ExitSuccess {
		t.Fatalf("curl failed: %#v", result)
	}
	if stdout.String() != "payload" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestCurlWritesToStdoutWithDashOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	result := New(nil).Run(context.Background(), cli.Invocation{
		Args:   []string{"-o", "-", server.URL},
		Stdout: &stdout,
	})

	if result.ExitCode != cli.ExitSuccess {
		t.Fatalf("curl failed: %#v", result)
	}
	if stdout.String() != "payload" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}
