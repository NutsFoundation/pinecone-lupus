package core

import (
	"context"
	"io"

	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

type GenericObjectOperations struct {
	parser   *URIParser
	resolver *EndpointResolver
	engine   clientarch.ObjectStorageEngine
}

var _ clientarch.ObjectOperations = (*GenericObjectOperations)(nil)

func NewGenericObjectOperations(parser *URIParser, resolver *EndpointResolver, engine clientarch.ObjectStorageEngine) *GenericObjectOperations {
	return &GenericObjectOperations{
		parser:   parser,
		resolver: resolver,
		engine:   engine,
	}
}

func (o *GenericObjectOperations) Get(ctx context.Context, uri string, opts clientarch.GetOptions) (clientarch.ObjectStream, error) {
	ref, err := o.toObjectRef(uri)
	if err != nil {
		return nil, err
	}
	return o.engine.GetObject(ctx, ref, opts)
}

func (o *GenericObjectOperations) Stat(ctx context.Context, uri string) (clientarch.ObjectMetadata, error) {
	ref, err := o.toObjectRef(uri)
	if err != nil {
		return clientarch.ObjectMetadata{}, err
	}
	return o.engine.StatObject(ctx, ref)
}

func (o *GenericObjectOperations) Put(ctx context.Context, uri string, body io.Reader, size int64, opts clientarch.PutOptions) (clientarch.PutResult, error) {
	ref, err := o.toObjectRef(uri)
	if err != nil {
		return clientarch.PutResult{}, err
	}
	return o.engine.PutObject(ctx, ref, body, size, opts)
}

func (o *GenericObjectOperations) Delete(ctx context.Context, uri string) error {
	ref, err := o.toObjectRef(uri)
	if err != nil {
		return err
	}
	return o.engine.DeleteObject(ctx, ref)
}

func (o *GenericObjectOperations) List(ctx context.Context, uri string, opts clientarch.ListOptions) (clientarch.ObjectListResult, error) {
	ref, err := o.toBucketRef(uri)
	if err != nil {
		return clientarch.ObjectListResult{}, err
	}
	return o.engine.ListObjects(ctx, ref, opts)
}

func (o *GenericObjectOperations) toObjectRef(raw string) (clientarch.ObjectRef, error) {
	uri, err := o.parser.Parse(raw)
	if err != nil {
		return clientarch.ObjectRef{}, err
	}
	endpoint, err := o.resolver.Resolve(uri)
	if err != nil {
		return clientarch.ObjectRef{}, err
	}
	return clientarch.ObjectRef{
		Endpoint:     endpoint,
		Bucket:       uri.Bucket,
		Key:          uri.Key,
		ResourceType: uri.ResourceType,
	}, nil
}

func (o *GenericObjectOperations) toBucketRef(raw string) (clientarch.BucketRef, error) {
	uri, err := o.parser.Parse(raw)
	if err != nil {
		return clientarch.BucketRef{}, err
	}
	endpoint, err := o.resolver.Resolve(uri)
	if err != nil {
		return clientarch.BucketRef{}, err
	}
	return clientarch.BucketRef{
		Endpoint: endpoint,
		Bucket:   uri.Bucket,
		Prefix:   uri.Key,
	}, nil
}
