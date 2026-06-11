package tar

type tarMode string

const (
	modeCreate  tarMode = "create"
	modeExtract tarMode = "extract"
	modeList    tarMode = "list"
)

type options struct {
	mode        tarMode
	archivePath string
	sourcePaths []string
	destDir     string
	gzip        bool
	overwrite   bool
}
