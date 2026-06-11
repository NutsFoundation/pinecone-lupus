package arch

import "context"

type KernelOperations interface {
	Read(ctx context.Context, path string) (ObjectStream, error)
	ReadURI(ctx context.Context, uri string) (ObjectStream, error)
}
