package http

import "io"

type DownloadConfig struct {
	DefaultTimeoutMillis        int64
	DefaultConnectTimeoutMillis int64
	MaxRedirects                int
	UserAgent                   string
	MaxBytes                    int64
}

type DownloadRequest struct {
	URL                  string
	OutputPath           string
	Output               io.Writer
	Headers              map[string]string
	FollowRedirects      bool
	FailOnHTTPError      bool
	TimeoutMillis        int64
	ConnectTimeoutMillis int64
	MaxBytes             int64
}
