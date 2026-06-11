package zip

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/archive"
	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

type Zip struct{}

func New() *Zip {
	return &Zip{}
}

func (z *Zip) Name() string {
	return "zip"
}

func (z *Zip) Run(ctx context.Context, invocation cli.Invocation) cli.Result {
	opts, err := parseRecover(invocation.Args)
	if err != nil {
		writeLine(invocation.Stderr, "zip: "+err.Error())
		return cli.Failure(cli.ExitArgumentError, err.Error())
	}

	result, err := archive.CreateZip(ctx, archive.ZipCreateOptions{
		OutputPath:  opts.outputPath,
		SourcePaths: opts.sourcePaths,
		Recursive:   opts.recursive,
	})
	if err != nil {
		writeLine(invocation.Stderr, "zip: "+err.Error())
		return cli.Failure(cli.ExitGenericError, err.Error())
	}
	if !opts.quiet {
		for _, entry := range result.Entries {
			writeLine(invocation.Stdout, fmt.Sprintf("%s %s", entry.Action, entry.Name))
		}
	}
	return cli.Success(fmt.Sprintf("added %d entries", len(result.Entries)))
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
