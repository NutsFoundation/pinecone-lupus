package minio

import (
	"context"
	"io"
	"net/url"
	"path"
	"strings"
	"sync"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	redprotocol "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/red/protocol"
	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
	clientcore "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/core"
)

type Engine struct {
	config  clientarch.Config
	mu      sync.Mutex
	clients map[string]*miniosdk.Client
}

var _ clientarch.ObjectStorageEngine = (*Engine)(nil)

func NewEngine(config clientarch.Config) *Engine {
	return &Engine{
		config:  config,
		clients: map[string]*miniosdk.Client{},
	}
}

func NewClient(config clientarch.Config) (clientarch.Client, error) {
	engine := NewEngine(config)
	return clientcore.NewGenericClient(config, engine), nil
}

func (e *Engine) GetObject(ctx context.Context, ref clientarch.ObjectRef, opts clientarch.GetOptions) (clientarch.ObjectStream, error) {
	client, err := e.client(ref.Endpoint)
	if err != nil {
		return nil, err
	}
	object, err := client.GetObject(ctx, ref.Bucket, ref.Key, e.toGetOptions(opts))
	if err != nil {
		return nil, clientarch.EngineError{Operation: "get", Bucket: ref.Bucket, Key: ref.Key, Cause: err}
	}
	metadata := clientarch.ObjectMetadata{
		Bucket:       ref.Bucket,
		Key:          ref.Key,
		Name:         objectName(ref.Key),
		ResourceType: ref.ResourceType,
	}
	return clientcore.NewObjectStream(object, metadata), nil
}

func (e *Engine) StatObject(ctx context.Context, ref clientarch.ObjectRef) (clientarch.ObjectMetadata, error) {
	client, err := e.client(ref.Endpoint)
	if err != nil {
		return clientarch.ObjectMetadata{}, err
	}
	info, err := client.StatObject(ctx, ref.Bucket, ref.Key, miniosdk.StatObjectOptions{})
	if err != nil {
		return clientarch.ObjectMetadata{}, clientarch.EngineError{Operation: "stat", Bucket: ref.Bucket, Key: ref.Key, Cause: err}
	}
	return toMetadata(ref, info), nil
}

func (e *Engine) PutObject(ctx context.Context, ref clientarch.ObjectRef, body io.Reader, size int64, opts clientarch.PutOptions) (clientarch.PutResult, error) {
	client, err := e.client(ref.Endpoint)
	if err != nil {
		return clientarch.PutResult{}, err
	}
	info, err := client.PutObject(ctx, ref.Bucket, ref.Key, body, size, e.toPutOptions(opts))
	if err != nil {
		return clientarch.PutResult{}, clientarch.EngineError{Operation: "put", Bucket: ref.Bucket, Key: ref.Key, Cause: err}
	}
	return clientarch.PutResult{
		Bucket: info.Bucket,
		Key:    info.Key,
		ETag:   info.ETag,
	}, nil
}

func (e *Engine) DeleteObject(ctx context.Context, ref clientarch.ObjectRef) error {
	client, err := e.client(ref.Endpoint)
	if err != nil {
		return err
	}
	err = client.RemoveObject(ctx, ref.Bucket, ref.Key, miniosdk.RemoveObjectOptions{})
	if err != nil {
		return clientarch.EngineError{Operation: "delete", Bucket: ref.Bucket, Key: ref.Key, Cause: err}
	}
	return nil
}

func (e *Engine) ListObjects(ctx context.Context, ref clientarch.BucketRef, opts clientarch.ListOptions) (clientarch.ObjectListResult, error) {
	client, err := e.client(ref.Endpoint)
	if err != nil {
		return clientarch.ObjectListResult{}, err
	}
	prefix := ref.Prefix
	if opts.Prefix != "" {
		prefix = opts.Prefix
	}
	recursive := opts.Recursive
	if opts == (clientarch.ListOptions{}) {
		recursive = true
	}
	objectCh := client.ListObjects(ctx, ref.Bucket, miniosdk.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})
	result := clientarch.ObjectListResult{
		Bucket: ref.Bucket,
		Prefix: prefix,
	}
	for object := range objectCh {
		if object.Err != nil {
			return clientarch.ObjectListResult{}, clientarch.EngineError{Operation: "list", Bucket: ref.Bucket, Cause: object.Err}
		}
		result.Objects = append(result.Objects, toEntry(ref.Bucket, object))
	}
	return result, nil
}

func (e *Engine) client(endpoint string) (*miniosdk.Client, error) {
	normalized := e.normalizeEndpoint(endpoint)
	e.mu.Lock()
	defer e.mu.Unlock()
	if client, ok := e.clients[normalized]; ok {
		return client, nil
	}
	client, err := miniosdk.New(normalized, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(e.config.AccessKey, e.config.SecretKey, ""),
		Secure: e.isSecure(endpoint),
		Region: e.config.Region,
	})
	if err != nil {
		return nil, err
	}
	e.clients[normalized] = client
	return client, nil
}

func (e *Engine) toGetOptions(opts clientarch.GetOptions) miniosdk.GetObjectOptions {
	var ret miniosdk.GetObjectOptions
	if opts.Range.Partial {
		_ = ret.SetRange(opts.Range.Start, opts.Range.Start+opts.Range.Length-1)
	}
	return ret
}

func (e *Engine) toPutOptions(opts clientarch.PutOptions) miniosdk.PutObjectOptions {
	return miniosdk.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
		PartSize:     e.config.AffirmPartSize(),
	}
}

func (e *Engine) normalizeEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if strings.HasPrefix(trimmed, redprotocol.HTTPScheme+"://") || strings.HasPrefix(trimmed, redprotocol.HTTPSSecureScheme+"://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			return parsed.Host
		}
	}
	return trimmed
}

func (e *Engine) isSecure(endpoint string) bool {
	trimmed := strings.TrimSpace(endpoint)
	if strings.HasPrefix(trimmed, redprotocol.HTTPSSecureScheme+"://") {
		return true
	}
	if strings.HasPrefix(trimmed, redprotocol.HTTPScheme+"://") {
		return false
	}
	return e.config.Secure
}

func toMetadata(ref clientarch.ObjectRef, info miniosdk.ObjectInfo) clientarch.ObjectMetadata {
	return clientarch.ObjectMetadata{
		Bucket:       ref.Bucket,
		Key:          info.Key,
		Name:         objectName(info.Key),
		Size:         info.Size,
		ETag:         info.ETag,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
		ResourceType: ref.ResourceType,
	}
}

func toEntry(bucket string, info miniosdk.ObjectInfo) clientarch.ObjectEntry {
	return clientarch.ObjectEntry{
		ObjectMetadata: clientarch.ObjectMetadata{
			Bucket:       bucket,
			Key:          info.Key,
			Name:         objectName(info.Key),
			Size:         info.Size,
			ETag:         info.ETag,
			ContentType:  info.ContentType,
			LastModified: info.LastModified,
			ResourceType: clientarch.ResourceTypeObject,
		},
		Directory: info.Key != "" && strings.HasSuffix(info.Key, "/"),
	}
}

func objectName(key string) string {
	trimmed := strings.TrimRight(key, "/")
	if trimmed == "" {
		return ""
	}
	return path.Base(trimmed)
}
