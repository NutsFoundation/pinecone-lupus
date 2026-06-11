package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriterReplacesExistingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	writer, err := NewAtomicWriter(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("new")); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestAtomicWriterAbortRemovesTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "payload.bin")
	writer, err := NewAtomicWriter(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("partial")); err != nil {
		writer.Abort()
		t.Fatal(err)
	}
	writer.Abort()

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target should not exist after abort, stat err=%v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary files remain: %v", entries)
	}
}
