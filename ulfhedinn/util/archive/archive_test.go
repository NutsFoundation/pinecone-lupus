package archive

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestZipCreateExtractAndSkipExisting(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(root, "bundle.zip")
	createResult, err := CreateZip(context.Background(), ZipCreateOptions{
		OutputPath:  archivePath,
		SourcePaths: []string{sourceDir},
		Recursive:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(createResult.Entries) == 0 {
		t.Fatal("zip create result is empty")
	}

	dest := filepath.Join(root, "dest")
	extractResult, err := ExtractZip(context.Background(), ZipExtractOptions{
		SourcePath: archivePath,
		DestDir:    dest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(extractResult.Entries) == 0 {
		t.Fatal("zip extract result is empty")
	}
	assertFile(t, filepath.Join(dest, "source", "hello.txt"), "hello")

	if err := os.WriteFile(filepath.Join(dest, "source", "hello.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = ExtractZip(context.Background(), ZipExtractOptions{
		SourcePath: archivePath,
		DestDir:    dest,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dest, "source", "hello.txt"), "changed")

	_, err = ExtractZip(context.Background(), ZipExtractOptions{
		SourcePath: archivePath,
		DestDir:    dest,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dest, "source", "hello.txt"), "hello")
}

func TestTarGzCreateListAndExtract(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(root, "bundle.tar.gz")
	_, err := CreateTar(context.Background(), TarCreateOptions{
		OutputPath:  archivePath,
		SourcePaths: []string{sourceDir},
		Gzip:        true,
	})
	if err != nil {
		t.Fatal(err)
	}

	listResult, err := ListTar(context.Background(), archivePath, true)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEntry(listResult.Entries, "source/hello.txt") {
		t.Fatalf("tar list missing source/hello.txt: %#v", listResult.Entries)
	}

	dest := filepath.Join(root, "dest")
	_, err = ExtractTar(context.Background(), TarExtractOptions{
		SourcePath: archivePath,
		DestDir:    dest,
		Gzip:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dest, "source", "hello.txt"), "hello")
}

func TestZipExtractRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "evil.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = ExtractZip(context.Background(), ZipExtractOptions{
		SourcePath: archivePath,
		DestDir:    filepath.Join(root, "dest"),
	})
	if err == nil {
		t.Fatal("path traversal zip should fail")
	}
}

func assertFile(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s = %q, want %q", path, string(content), want)
	}
}

func hasEntry(entries []EntryResult, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return true
		}
	}
	return false
}
