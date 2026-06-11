package arch

import (
	"context"
	"io"
	"time"
)

type ObjectOperations interface {
	Get(ctx context.Context, uri string, opts GetOptions) (ObjectStream, error)
	Stat(ctx context.Context, uri string) (ObjectMetadata, error)
	Put(ctx context.Context, uri string, body io.Reader, size int64, opts PutOptions) (PutResult, error)
	Delete(ctx context.Context, uri string) error
	List(ctx context.Context, uri string, opts ListOptions) (ObjectListResult, error)
}

type ObjectStream interface {
	io.ReadCloser
	Metadata() ObjectMetadata
}

type ObjectRef struct {
	Endpoint     string
	Bucket       string
	Key          string
	ResourceType ResourceType
}

type BucketRef struct {
	Endpoint string
	Bucket   string
	Prefix   string
}

type ObjectMetadata struct {
	Bucket       string
	Key          string
	Name         string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
	ResourceType ResourceType
}

type ObjectEntry struct {
	ObjectMetadata
	Directory bool
}

type ObjectListResult struct {
	Bucket                string
	Prefix                string
	Objects               []ObjectEntry
	Truncated             bool
	NextContinuationToken string
}

type PutResult struct {
	Bucket string
	Key    string
	ETag   string
}
