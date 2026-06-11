package archive

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func safeExtractPath(destDir string, name string) (string, error) {
	if destDir == "" {
		return "", fmt.Errorf("destination directory is required")
	}
	if name == "" {
		return "", fmt.Errorf("archive entry name is empty")
	}
	if strings.Contains(name, "\\") {
		return "", fmt.Errorf("archive entry contains backslash path: %s", name)
	}
	if path.IsAbs(name) || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", fmt.Errorf("archive entry uses absolute path: %s", name)
	}
	cleanName := path.Clean(name)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("archive entry escapes destination: %s", name)
	}

	root := filepath.Clean(destDir)
	target := filepath.Clean(filepath.Join(root, filepath.FromSlash(cleanName)))
	if !isInside(root, target) {
		return "", fmt.Errorf("archive entry escapes destination: %s", name)
	}
	return target, nil
}

func safeArchiveName(root string, target string) (string, error) {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes archive root: %s", target)
	}
	name := filepath.ToSlash(rel)
	if name == "." || name == "" {
		return "", fmt.Errorf("archive name is empty: %s", target)
	}
	return name, nil
}

func isInside(root string, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(target, prefix)
}

func ensureRegularOrDir(info os.FileInfo, name string) error {
	mode := info.Mode()
	if mode.IsRegular() || mode.IsDir() {
		return nil
	}
	return fmt.Errorf("unsupported file type for archive entry: %s", name)
}
