package fs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type AtomicWriter struct {
	targetPath string
	tempPath   string
	file       *os.File
	committed  bool
}

func NewAtomicWriter(targetPath string) (*AtomicWriter, error) {
	if targetPath == "" {
		return nil, fmt.Errorf("target path is required")
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	tempPath := filepath.Join(dir, fmt.Sprintf(".%s.ulf-download-%s.tmp", filepath.Base(targetPath), randomSuffix()))
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &AtomicWriter{targetPath: targetPath, tempPath: tempPath, file: file}, nil
}

func (w *AtomicWriter) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

func (w *AtomicWriter) Commit() error {
	if w.committed {
		return nil
	}
	if err := w.file.Close(); err != nil {
		_ = os.Remove(w.tempPath)
		return err
	}
	if err := os.Rename(w.tempPath, w.targetPath); err != nil {
		if removeErr := os.Remove(w.targetPath); removeErr != nil && !os.IsNotExist(removeErr) {
			_ = os.Remove(w.tempPath)
			return err
		}
		if renameErr := os.Rename(w.tempPath, w.targetPath); renameErr != nil {
			_ = os.Remove(w.tempPath)
			return renameErr
		}
	}
	w.committed = true
	return nil
}

func (w *AtomicWriter) Abort() {
	if w == nil || w.committed {
		return
	}
	_ = w.file.Close()
	_ = os.Remove(w.tempPath)
}

func randomSuffix() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(buf)
}
