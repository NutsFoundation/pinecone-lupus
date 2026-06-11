package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ulffs "github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/fs"
)

type TarExtractOptions struct {
	SourcePath string
	DestDir    string
	Gzip       bool
	Overwrite  bool
}

type TarCreateOptions struct {
	OutputPath  string
	SourcePaths []string
	Gzip        bool
}

func ListTar(ctx context.Context, sourcePath string, gzipMode bool) (ListResult, error) {
	file, reader, closeFn, err := openTarReader(sourcePath, gzipMode)
	if err != nil {
		return ListResult{}, err
	}
	defer file.Close()
	defer closeFn()

	result := ListResult{}
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		header, err := reader.Next()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, EntryResult{Name: header.Name, Size: header.Size})
	}
}

func ExtractTar(ctx context.Context, opts TarExtractOptions) (ExtractResult, error) {
	file, reader, closeFn, err := openTarReader(opts.SourcePath, opts.Gzip)
	if err != nil {
		return ExtractResult{}, err
	}
	defer file.Close()
	defer closeFn()

	if opts.DestDir == "" {
		opts.DestDir = "."
	}
	if err := os.MkdirAll(opts.DestDir, 0755); err != nil {
		return ExtractResult{}, err
	}

	result := ExtractResult{}
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		header, err := reader.Next()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		target, err := safeExtractPath(opts.DestDir, header.Name)
		if err != nil {
			return result, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return result, err
			}
			result.Entries = append(result.Entries, EntryResult{Name: header.Name, Path: target, Action: EntryDirectory})
		case tar.TypeReg, tar.TypeRegA:
			if _, err := os.Stat(target); err == nil && !opts.Overwrite {
				result.Entries = append(result.Entries, EntryResult{Name: header.Name, Path: target, Size: header.Size, Action: EntrySkipped})
				continue
			} else if err != nil && !os.IsNotExist(err) {
				return result, err
			}
			if err := extractTarFile(reader, target); err != nil {
				return result, err
			}
			result.Entries = append(result.Entries, EntryResult{Name: header.Name, Path: target, Size: header.Size, Action: EntryExtracted})
		default:
			return result, fmt.Errorf("unsupported tar entry type for %s", header.Name)
		}
	}
}

func CreateTar(ctx context.Context, opts TarCreateOptions) (CreateResult, error) {
	if opts.OutputPath == "" {
		return CreateResult{}, fmt.Errorf("tar output path is required")
	}
	if len(opts.SourcePaths) == 0 {
		return CreateResult{}, fmt.Errorf("tar source path is required")
	}

	writer, err := ulffs.NewAtomicWriter(opts.OutputPath)
	if err != nil {
		return CreateResult{}, err
	}
	defer writer.Abort()

	var closeFn func() error
	var tarWriter *tar.Writer
	if opts.Gzip {
		gzipWriter := gzip.NewWriter(writer)
		tarWriter = tar.NewWriter(gzipWriter)
		closeFn = func() error {
			if err := tarWriter.Close(); err != nil {
				_ = gzipWriter.Close()
				return err
			}
			return gzipWriter.Close()
		}
	} else {
		tarWriter = tar.NewWriter(writer)
		closeFn = tarWriter.Close
	}

	result := CreateResult{}
	for _, source := range opts.SourcePaths {
		if err := ctx.Err(); err != nil {
			_ = closeFn()
			return result, err
		}
		if err := addTarSource(ctx, tarWriter, filepath.Clean(source), &result); err != nil {
			_ = closeFn()
			return result, err
		}
	}
	if err := closeFn(); err != nil {
		return result, err
	}
	return result, writer.Commit()
}

func openTarReader(sourcePath string, gzipMode bool) (*os.File, *tar.Reader, func() error, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, nil, nil, err
	}
	if gzipMode {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			_ = file.Close()
			return nil, nil, nil, err
		}
		return file, tar.NewReader(gzipReader), gzipReader.Close, nil
	}
	return file, tar.NewReader(file), func() error { return nil }, nil
}

func extractTarFile(reader *tar.Reader, target string) error {
	writer, err := ulffs.NewAtomicWriter(target)
	if err != nil {
		return err
	}
	defer writer.Abort()
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	return writer.Commit()
}

func addTarSource(ctx context.Context, tarWriter *tar.Writer, source string, result *CreateResult) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if err := ensureRegularOrDir(info, source); err != nil {
		return err
	}
	root := filepath.Dir(source)
	if info.IsDir() {
		return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if path == source {
				return nil
			}
			if err := ensureRegularOrDir(info, path); err != nil {
				return err
			}
			return writeTarEntry(tarWriter, root, path, info, result)
		})
	}
	return writeTarEntry(tarWriter, root, source, info, result)
}

func writeTarEntry(tarWriter *tar.Writer, root string, source string, info os.FileInfo, result *CreateResult) error {
	name, err := safeArchiveName(root, source)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if info.IsDir() {
		result.Entries = append(result.Entries, EntryResult{Name: header.Name, Path: source, Action: EntryDirectory})
		return nil
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(tarWriter, file); err != nil {
		return err
	}
	result.Entries = append(result.Entries, EntryResult{Name: header.Name, Path: source, Size: info.Size(), Action: EntryAdded})
	return nil
}
