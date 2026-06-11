package arch

type ResourceType string

const (
	ResourceTypeObject ResourceType = "Object"
	ResourceTypeKernel ResourceType = "Kernel"
)

type ShuttleURI struct {
	Raw              string
	Scheme           string
	Endpoint         string
	Bucket           string
	Key              string
	Path             string
	ExplicitEndpoint bool
	ResourceType     ResourceType
}
