package core

import (
	"context"
	"strings"

	redprotocol "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/red/protocol"
	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

type GenericKernelOperations struct {
	config  clientarch.Config
	objects clientarch.ObjectOperations
}

var _ clientarch.KernelOperations = (*GenericKernelOperations)(nil)

func NewGenericKernelOperations(config clientarch.Config, objects clientarch.ObjectOperations) *GenericKernelOperations {
	return &GenericKernelOperations{
		config:  config,
		objects: objects,
	}
}

func (o *GenericKernelOperations) Read(ctx context.Context, path string) (clientarch.ObjectStream, error) {
	return o.objects.Get(ctx, o.toSystemURI(path), clientarch.GetOptions{})
}

func (o *GenericKernelOperations) ReadURI(ctx context.Context, uri string) (clientarch.ObjectStream, error) {
	return o.objects.Get(ctx, uri, clientarch.GetOptions{})
}

func (o *GenericKernelOperations) toSystemURI(path string) string {
	body := strings.TrimLeft(path, "/")
	return redprotocol.SchemeRed + "://" + o.config.AffirmSystemBucket() + "/" + body
}
