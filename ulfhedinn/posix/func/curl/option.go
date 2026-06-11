package curl

type options struct {
	url                  string
	outputPath           string
	remoteName           bool
	failOnHTTPError      bool
	silent               bool
	showError            bool
	followRedirects      bool
	headers              map[string]string
	timeoutMillis        int64
	connectTimeoutMillis int64
}
