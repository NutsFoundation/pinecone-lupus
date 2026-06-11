package unzip

type options struct {
	sourcePath string
	destDir    string
	overwrite  bool
	quiet      bool
	listOnly   bool
}
