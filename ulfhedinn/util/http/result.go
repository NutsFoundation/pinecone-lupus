package http

type DownloadResult struct {
	Success      bool
	ExitCode     int
	StatusCode   int
	URL          string
	FinalURL     string
	OutputPath   string
	BytesWritten int64
	Message      string
	Error        error
}
