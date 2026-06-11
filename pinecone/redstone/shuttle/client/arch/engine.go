package arch

import (
	"context"
	"io"
)

type ObjectStorageEngine interface {
	GetObject(ctx context.Context, ref ObjectRef, opts GetOptions) (ObjectStream, error)
	StatObject(ctx context.Context, ref ObjectRef) (ObjectMetadata, error)
	PutObject(ctx context.Context, ref ObjectRef, body io.Reader, size int64, opts PutOptions) (PutResult, error)
	DeleteObject(ctx context.Context, ref ObjectRef) error
	ListObjects(ctx context.Context, ref BucketRef, opts ListOptions) (ObjectListResult, error)
}
