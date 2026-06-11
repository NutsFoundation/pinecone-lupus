package core

import (
	"strings"

	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

type EndpointResolver struct {
	config clientarch.Config
}

func NewEndpointResolver(config clientarch.Config) *EndpointResolver {
	return &EndpointResolver{config: config}
}

func (r *EndpointResolver) Resolve(uri clientarch.ShuttleURI) (string, error) {
	if strings.TrimSpace(uri.Endpoint) != "" {
		return uri.Endpoint, nil
	}
	if strings.TrimSpace(r.config.DefaultEndpoint) != "" {
		return r.config.DefaultEndpoint, nil
	}
	return "", clientarch.URIError{Message: "red shuttle endpoint is not configured"}
}
