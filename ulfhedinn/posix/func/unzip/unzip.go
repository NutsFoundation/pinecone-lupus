package unzip

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/archive"
	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

type Unzip struct{}

func New() *Unzip {
	return &Unzip{}
}

func (u *Unzip) Name() string {
	return "unzip"
}

func (u *Unzip) Run(ctx context.Context, invocation cli.Invocation) cli.Result {
	opts, err := parseRecover(invocation.Args)
	if err != nil {
		writeLine(invocation.Stderr, "unzip: "+err.Error())
		return cli.Failure(cli.ExitArgumentError, err.Error())
	}

	if opts.listOnly {
		result, err := archive.ListZip(ctx, opts.sourcePath)
		if err != nil {
			writeLine(invocation.Stderr, "unzip: "+err.Error())
			return cli.Failure(cli.ExitGenericError, err.Error())
		}
		if !opts.quiet {
			for _, entry := range result.Entries {
				writeLine(invocation.Stdout, entry.Name)
			}
		}
		return cli.Success(fmt.Sprintf("listed %d entries", len(result.Entries)))
	}

	result, err := archive.ExtractZip(ctx, archive.ZipExtractOptions{
		SourcePath: opts.sourcePath,
		DestDir:    opts.destDir,
		Overwrite:  opts.overwrite,
	})
	if err != nil {
		writeLine(invocation.Stderr, "unzip: "+err.Error())
		return cli.Failure(cli.ExitGenericError, err.Error())
	}
	if !opts.quiet {
		for _, entry := range result.Entries {
			writeLine(invocation.Stdout, fmt.Sprintf("%s %s", entry.Action, entry.Name))
		}
	}
	return cli.Success(fmt.Sprintf("processed %d entries", len(result.Entries)))
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
