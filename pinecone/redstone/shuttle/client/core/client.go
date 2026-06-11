package core

import clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"

type GenericClient struct {
	objects clientarch.ObjectOperations
	kernel  clientarch.KernelOperations
}

var _ clientarch.Client = (*GenericClient)(nil)

func NewGenericClient(config clientarch.Config, engine clientarch.ObjectStorageEngine) *GenericClient {
	parser := NewURIParser(config.AffirmSystemBucket())
	resolver := NewEndpointResolver(config)
	objects := NewGenericObjectOperations(parser, resolver, engine)
	return &GenericClient{
		objects: objects,
		kernel:  NewGenericKernelOperations(config, objects),
	}
}

func (c *GenericClient) Objects() clientarch.ObjectOperations {
	if c == nil {
		return nil
	}
	return c.objects
}

func (c *GenericClient) Kernel() clientarch.KernelOperations {
	if c == nil {
		return nil
	}
	return c.kernel
}
