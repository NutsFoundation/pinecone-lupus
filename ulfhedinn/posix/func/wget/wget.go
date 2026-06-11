package wget

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
	ulfhttp "github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/http"
)

type Wget struct {
	downloader *ulfhttp.Downloader
}

func New(downloader *ulfhttp.Downloader) *Wget {
	if downloader == nil {
		downloader = ulfhttp.NewDownloader(ulfhttp.DownloadConfig{})
	}
	return &Wget{downloader: downloader}
}

func (w *Wget) Name() string {
	return "wget"
}

func (w *Wget) Run(ctx context.Context, invocation cli.Invocation) cli.Result {
	opts, err := parseRecover(invocation.Args)
	if err != nil {
		writeLine(invocation.Stderr, "wget: "+err.Error())
		return cli.Failure(cli.ExitArgumentError, err.Error())
	}

	output := io.Writer(nil)
	outputPath := ""
	if opts.outputPath == "-" {
		output = invocation.Stdout
	} else {
		outputPath = resolvePath(invocation.WorkDir, opts.outputPath)
	}

	result := w.downloader.Download(ctx, ulfhttp.DownloadRequest{
		URL:             opts.url,
		OutputPath:      outputPath,
		Output:          output,
		Headers:         opts.headers,
		FollowRedirects: true,
		FailOnHTTPError: true,
		TimeoutMillis:   opts.timeoutMillis,
	})
	if !result.Success {
		if !opts.quiet {
			message := result.Message
			if result.Error != nil {
				message = fmt.Sprintf("%s: %v", message, result.Error)
			}
			writeLine(invocation.Stderr, "wget: "+message)
		}
		return cli.Failure(result.ExitCode, result.Message)
	}
	return cli.Success(result.Message)
}

func parseRecover(args []string) (opts options, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("%v", value)
		}
	}()
	return parse(args)
}

func resolvePath(workDir string, outputPath string) string {
	if outputPath == "" || filepath.IsAbs(outputPath) || workDir == "" {
		return outputPath
	}
	return filepath.Join(workDir, outputPath)
}

func writeLine(writer io.Writer, message string) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintln(writer, message)
}
