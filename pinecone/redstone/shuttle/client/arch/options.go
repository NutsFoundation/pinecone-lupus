package arch

type ObjectRange struct {
	Start   int64
	Length  int64
	Partial bool
}

type GetOptions struct {
	Range ObjectRange
}

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

type ListOptions struct {
	Prefix    string
	Recursive bool
}

func DefaultListOptions() ListOptions {
	return ListOptions{Recursive: true}
}
