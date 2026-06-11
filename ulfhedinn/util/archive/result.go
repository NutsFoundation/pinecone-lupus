package archive

type EntryAction string

const (
	EntryExtracted EntryAction = "extracted"
	EntrySkipped   EntryAction = "skipped"
	EntryDirectory EntryAction = "directory"
	EntryAdded     EntryAction = "added"
)

type EntryResult struct {
	Name   string
	Path   string
	Size   int64
	Action EntryAction
}

type ExtractResult struct {
	Entries []EntryResult
}

type ListResult struct {
	Entries []EntryResult
}

type CreateResult struct {
	Entries []EntryResult
}
