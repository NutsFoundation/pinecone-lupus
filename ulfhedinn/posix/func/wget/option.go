package wget

type options struct {
	url           string
	outputPath    string
	quiet         bool
	headers       map[string]string
	timeoutMillis int64
}
