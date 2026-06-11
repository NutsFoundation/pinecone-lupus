package curl

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
	ulfhttp "github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/http"
)

type Curl struct {
	downloader *ulfhttp.Downloader
}

func New(downloader *ulfhttp.Downloader) *Curl {
	if downloader == nil {
		downloader = ulfhttp.NewDownloader(ulfhttp.DownloadConfig{})
	}
	return &Curl{downloader: downloader}
}

func (c *Curl) Name() string {
	return "curl"
}

func (c *Curl) Run(ctx context.Context, invocation cli.Invocation) cli.Result {
	opts, err := parseRecover(invocation.Args)
	if err != nil {
		writeLine(invocation.Stderr, "curl: "+err.Error())
		return cli.Failure(cli.ExitArgumentError, err.Error())
	}

	output := invocation.Stdout
	outputPath := ""
	if opts.outputPath != "-" {
		outputPath = resolvePath(invocation.WorkDir, opts.outputPath)
	}
	if outputPath != "" {
		output = nil
	}
	result := c.downloader.Download(ctx, ulfhttp.DownloadRequest{
		URL:                  opts.url,
		OutputPath:           outputPath,
		Output:               output,
		Headers:              opts.headers,
		FollowRedirects:      opts.followRedirects,
		FailOnHTTPError:      opts.failOnHTTPError,
		TimeoutMillis:        opts.timeoutMillis,
		ConnectTimeoutMillis: opts.connectTimeoutMillis,
	})
	if !result.Success {
		if !opts.silent || opts.showError {
			message := result.Message
			if result.Error != nil {
				message = fmt.Sprintf("%s: %v", message, result.Error)
			}
			writeLine(invocation.Stderr, "curl: "+message)
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
