package http

import (
	"context"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/fs"
)

type Downloader struct {
	config DownloadConfig
	client *stdhttp.Client
}

func NewDownloader(config DownloadConfig) *Downloader {
	connectTimeout := durationMillis(config.DefaultConnectTimeoutMillis)
	if connectTimeout <= 0 {
		connectTimeout = 30 * time.Second
	}
	transport := &stdhttp.Transport{
		Proxy: stdhttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
	}
	return &Downloader{
		config: config,
		client: &stdhttp.Client{Transport: transport},
	}
}

func (d *Downloader) Download(ctx context.Context, request DownloadRequest) DownloadResult {
	result := DownloadResult{URL: request.URL, OutputPath: request.OutputPath}
	parsed, err := url.Parse(request.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return failed(result, cli.ExitArgumentError, "invalid URL", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return failed(result, cli.ExitUnsupported, "unsupported URL scheme: "+parsed.Scheme, nil)
	}
	if request.OutputPath == "" && request.Output == nil {
		return failed(result, cli.ExitArgumentError, "output path or output writer is required", nil)
	}

	timeout := durationMillis(firstPositive(request.TimeoutMillis, d.config.DefaultTimeoutMillis))
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	httpClient := *d.client
	if request.ConnectTimeoutMillis > 0 {
		httpClient.Transport = transportWithConnectTimeout(durationMillis(request.ConnectTimeoutMillis))
	}
	if !request.FollowRedirects {
		httpClient.CheckRedirect = func(req *stdhttp.Request, via []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		}
	} else if d.config.MaxRedirects > 0 {
		httpClient.CheckRedirect = func(req *stdhttp.Request, via []*stdhttp.Request) error {
			if len(via) >= d.config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", len(via))
			}
			return nil
		}
	}

	httpReq, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodGet, request.URL, nil)
	if err != nil {
		return failed(result, cli.ExitArgumentError, "request creation failed", err)
	}
	userAgent := d.config.UserAgent
	if userAgent == "" {
		userAgent = "ulfhedinn/0.1"
	}
	httpReq.Header.Set("User-Agent", userAgent)
	for key, value := range request.Headers {
		httpReq.Header.Set(key, value)
	}

	response, err := httpClient.Do(httpReq)
	if err != nil {
		return failed(result, classifyError(err), "request failed", err)
	}
	defer response.Body.Close()

	result.StatusCode = response.StatusCode
	if response.Request != nil && response.Request.URL != nil {
		result.FinalURL = response.Request.URL.String()
	}
	if result.FinalURL == "" {
		result.FinalURL = request.URL
	}

	if request.FailOnHTTPError && response.StatusCode >= 400 {
		return failed(result, cli.ExitHTTPFailure, fmt.Sprintf("HTTP request failed: %d", response.StatusCode), nil)
	}

	writer := request.Output
	var atomicWriter *fs.AtomicWriter
	if request.OutputPath != "" {
		targetPath := request.OutputPath
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Clean(targetPath)
		}
		atomicWriter, err = fs.NewAtomicWriter(targetPath)
		if err != nil {
			return failed(result, cli.ExitWriteFailure, "failed to create output file", err)
		}
		defer atomicWriter.Abort()
		writer = atomicWriter
	}

	reader := response.Body
	maxBytes := firstPositive(request.MaxBytes, d.config.MaxBytes)
	if maxBytes > 0 {
		reader = io.NopCloser(io.LimitReader(response.Body, maxBytes+1))
	}
	bytesWritten, err := io.Copy(writer, reader)
	if err != nil {
		return failed(result, classifyError(err), "download failed", err)
	}
	if maxBytes > 0 && bytesWritten > maxBytes {
		return failed(result, cli.ExitWriteFailure, "download exceeded max bytes", nil)
	}
	result.BytesWritten = bytesWritten

	if atomicWriter != nil {
		if err := atomicWriter.Commit(); err != nil {
			return failed(result, cli.ExitWriteFailure, "failed to commit output file", err)
		}
	}

	result.Success = true
	result.ExitCode = cli.ExitSuccess
	result.Message = "download completed"
	return result
}

func transportWithConnectTimeout(connectTimeout time.Duration) *stdhttp.Transport {
	if connectTimeout <= 0 {
		connectTimeout = 30 * time.Second
	}
	return &stdhttp.Transport{
		Proxy: stdhttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
	}
}

func failed(result DownloadResult, exitCode int, message string, err error) DownloadResult {
	if exitCode == cli.ExitSuccess {
		exitCode = cli.ExitGenericError
	}
	result.Success = false
	result.ExitCode = exitCode
	result.Message = message
	result.Error = err
	return result
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func durationMillis(value int64) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}
