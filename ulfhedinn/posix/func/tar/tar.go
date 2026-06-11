package tar

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/archive"
	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

type Tar struct{}

func New() *Tar {
	return &Tar{}
}

func (t *Tar) Name() string {
	return "tar"
}

func (t *Tar) Run(ctx context.Context, invocation cli.Invocation) cli.Result {
	opts, err := parseRecover(invocation.Args)
	if err != nil {
		writeLine(invocation.Stderr, "tar: "+err.Error())
		return cli.Failure(cli.ExitArgumentError, err.Error())
	}

	switch opts.mode {
	case modeList:
		result, err := archive.ListTar(ctx, opts.archivePath, opts.gzip)
		if err != nil {
			writeLine(invocation.Stderr, "tar: "+err.Error())
			return cli.Failure(cli.ExitGenericError, err.Error())
		}
		for _, entry := range result.Entries {
			writeLine(invocation.Stdout, entry.Name)
		}
		return cli.Success(fmt.Sprintf("listed %d entries", len(result.Entries)))
	case modeExtract:
		result, err := archive.ExtractTar(ctx, archive.TarExtractOptions{
			SourcePath: opts.archivePath,
			DestDir:    opts.destDir,
			Gzip:       opts.gzip,
			Overwrite:  opts.overwrite,
		})
		if err != nil {
			writeLine(invocation.Stderr, "tar: "+err.Error())
			return cli.Failure(cli.ExitGenericError, err.Error())
		}
		return cli.Success(fmt.Sprintf("processed %d entries", len(result.Entries)))
	case modeCreate:
		result, err := archive.CreateTar(ctx, archive.TarCreateOptions{
			OutputPath:  opts.archivePath,
			SourcePaths: opts.sourcePaths,
			Gzip:        opts.gzip,
		})
		if err != nil {
			writeLine(invocation.Stderr, "tar: "+err.Error())
			return cli.Failure(cli.ExitGenericError, err.Error())
		}
		return cli.Success(fmt.Sprintf("added %d entries", len(result.Entries)))
	default:
		return cli.Failure(cli.ExitArgumentError, "tar mode is required")
	}
}

func parseRecover(args []string) (opts options, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("%v", value)
		}
	}()
	return parse(args)
}

func writeLine(writer io.Writer, message string) {
	if writer == nil || strings.TrimSpace(message) == "" {
		return
	}
	_, _ = fmt.Fprintln(writer, message)
}
