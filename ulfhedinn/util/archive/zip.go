package archive

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ulffs "github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/fs"
)

type ZipExtractOptions struct {
	SourcePath string
	DestDir    string
	Overwrite  bool
}

type ZipCreateOptions struct {
	OutputPath  string
	SourcePaths []string
	Recursive   bool
}

func ListZip(_ context.Context, sourcePath string) (ListResult, error) {
	reader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return ListResult{}, err
	}
	defer reader.Close()

	result := ListResult{}
	for _, file := range reader.File {
		result.Entries = append(result.Entries, EntryResult{
			Name: file.Name,
			Size: int64(file.UncompressedSize64),
		})
	}
	return result, nil
}

func ExtractZip(ctx context.Context, opts ZipExtractOptions) (ExtractResult, error) {
	reader, err := zip.OpenReader(opts.SourcePath)
	if err != nil {
		return ExtractResult{}, err
	}
	defer reader.Close()

	if opts.DestDir == "" {
		opts.DestDir = "."
	}
	if err := os.MkdirAll(opts.DestDir, 0755); err != nil {
		return ExtractResult{}, err
	}

	result := ExtractResult{}
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		target, err := safeExtractPath(opts.DestDir, file.Name)
		if err != nil {
			return result, err
		}
		mode := file.FileInfo().Mode()
		if mode&os.ModeSymlink != 0 {
			return result, fmt.Errorf("symlink archive entry is not supported: %s", file.Name)
		}
		if file.FileInfo().IsDir() || strings.HasSuffix(file.Name, "/") {
			if err := os.MkdirAll(target, 0755); err != nil {
				return result, err
			}
			result.Entries = append(result.Entries, EntryResult{Name: file.Name, Path: target, Action: EntryDirectory})
			continue
		}
		if !mode.IsRegular() {
			return result, fmt.Errorf("unsupported zip entry type: %s", file.Name)
		}
		if _, err := os.Stat(target); err == nil && !opts.Overwrite {
			result.Entries = append(result.Entries, EntryResult{Name: file.Name, Path: target, Size: int64(file.UncompressedSize64), Action: EntrySkipped})
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return result, err
		}
		if err := extractZipFile(file, target); err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, EntryResult{Name: file.Name, Path: target, Size: int64(file.UncompressedSize64), Action: EntryExtracted})
	}
	return result, nil
}

func extractZipFile(file *zip.File, target string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

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

func CreateZip(ctx context.Context, opts ZipCreateOptions) (CreateResult, error) {
	if opts.OutputPath == "" {
		return CreateResult{}, fmt.Errorf("zip output path is required")
	}
	if len(opts.SourcePaths) == 0 {
		return CreateResult{}, fmt.Errorf("zip source path is required")
	}

	writer, err := ulffs.NewAtomicWriter(opts.OutputPath)
	if err != nil {
		return CreateResult{}, err
	}
	defer writer.Abort()
	zipWriter := zip.NewWriter(writer)

	result := CreateResult{}
	for _, source := range opts.SourcePaths {
		if err := ctx.Err(); err != nil {
			_ = zipWriter.Close()
			return result, err
		}
		if err := addZipSource(ctx, zipWriter, filepath.Clean(source), opts.Recursive, &result); err != nil {
			_ = zipWriter.Close()
			return result, err
		}
	}
	if err := zipWriter.Close(); err != nil {
		return result, err
	}
	return result, writer.Commit()
}

func addZipSource(ctx context.Context, zipWriter *zip.Writer, source string, recursive bool, result *CreateResult) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if err := ensureRegularOrDir(info, source); err != nil {
		return err
	}
	root := filepath.Dir(source)
	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("source is a directory; use -r: %s", source)
		}
		root = filepath.Dir(source)
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
			return writeZipEntry(zipWriter, root, path, info, result)
		})
	}
	return writeZipEntry(zipWriter, root, source, info, result)
}

func writeZipEntry(zipWriter *zip.Writer, root string, source string, info os.FileInfo, result *CreateResult) error {
	name, err := safeArchiveName(root, source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if !strings.HasSuffix(name, "/") {
			name += "/"
		}
		_, err := zipWriter.Create(name)
		if err == nil {
			result.Entries = append(result.Entries, EntryResult{Name: name, Path: source, Action: EntryDirectory})
		}
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate
	entry, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(entry, file); err != nil {
		return err
	}
	result.Entries = append(result.Entries, EntryResult{Name: name, Path: source, Size: info.Size(), Action: EntryAdded})
	return nil
}
