package core

import (
	"net/url"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/imperium"
	redprotocol "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/red/protocol"
	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

type URIParser struct {
	systemBucket string
}

func NewURIParser(systemBucket string) *URIParser {
	if strings.TrimSpace(systemBucket) == "" {
		systemBucket = clientarch.DefaultSystemBucket
	}
	return &URIParser{
		systemBucket: systemBucket,
	}
}

func (p *URIParser) Parse(raw string) (clientarch.ShuttleURI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return clientarch.ShuttleURI{}, clientarch.URIError{Message: "red shuttle uri is blank"}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return clientarch.ShuttleURI{}, err
	}
	if parsed.Scheme == "" {
		return p.parsePath(raw), nil
	}
	scheme := strings.ToLower(parsed.Scheme)
	if !redprotocol.IsSupportedScheme(scheme) {
		return clientarch.ShuttleURI{}, clientarch.URIError{Message: "unsupported red shuttle uri scheme: " + scheme}
	}

	ret := clientarch.ShuttleURI{
		Raw:          raw,
		Scheme:       scheme,
		Path:         normalizePath(parsed.Path),
		ResourceType: clientarch.ResourceTypeObject,
	}
	p.fillAuthorityURI(&ret, authorityOf(parsed), ret.Path)
	p.markResourceType(&ret)
	return ret, nil
}

func (p *URIParser) parsePath(path string) clientarch.ShuttleURI {
	ret := clientarch.ShuttleURI{
		Raw:          path,
		Scheme:       redprotocol.SchemeRed,
		Path:         normalizePath(path),
		ResourceType: clientarch.ResourceTypeObject,
	}
	p.fillPathURI(&ret, ret.Path)
	p.markResourceType(&ret)
	return ret
}

func (p *URIParser) fillAuthorityURI(ret *clientarch.ShuttleURI, authority, path string) {
	if strings.TrimSpace(authority) == "" {
		ret.Bucket = p.systemBucket
		ret.Key = trimLeftSlash(path)
		ret.ResourceType = clientarch.ResourceTypeKernel
		return
	}
	if isEndpointAuthority(authority) {
		ret.ExplicitEndpoint = true
		ret.Endpoint = authority
		p.fillPathURI(ret, path)
		return
	}
	ret.Bucket = authority
	ret.Key = trimLeftSlash(path)
}

func (p *URIParser) fillPathURI(ret *clientarch.ShuttleURI, path string) {
	body := trimLeftSlash(normalizePath(path))
	slash := strings.IndexByte(body, '/')
	if slash < 0 {
		ret.Bucket = body
		ret.Key = ""
		return
	}
	ret.Bucket = body[:slash]
	ret.Key = body[slash+1:]
}

func (p *URIParser) markResourceType(uri *clientarch.ShuttleURI) {
	if uri.Bucket == p.systemBucket {
		uri.ResourceType = clientarch.ResourceTypeKernel
		return
	}
	if strings.TrimSpace(uri.Bucket) == "" && p.isKernelKey(uri.Key) {
		uri.Bucket = p.systemBucket
		uri.ResourceType = clientarch.ResourceTypeKernel
		return
	}
	if p.isKernelKey(uri.Bucket) {
		uri.Key = joinKey(uri.Bucket, uri.Key)
		uri.Bucket = p.systemBucket
		uri.ResourceType = clientarch.ResourceTypeKernel
	}
}

func (p *URIParser) isKernelKey(value string) bool {
	body := trimLeftSlash(value)
	if body == "" {
		return false
	}
	slash := strings.IndexByte(body, '/')
	root := body
	if slash >= 0 {
		root = body[:slash]
	}
	return imperium.IsKernelRoot(root)
}

func isEndpointAuthority(authority string) bool {
	if strings.TrimSpace(authority) == "" {
		return false
	}
	if strings.Contains(authority, "@") {
		return false
	}
	return strings.Contains(authority, ":") || strings.EqualFold(authority, "localhost") || strings.Contains(authority, ".")
}

func authorityOf(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	if parsed.User == nil {
		return parsed.Host
	}
	return parsed.User.String() + "@" + parsed.Host
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func trimLeftSlash(value string) string {
	return strings.TrimLeft(value, "/")
}

func joinKey(head, tail string) string {
	if strings.TrimSpace(tail) == "" {
		return trimLeftSlash(head)
	}
	return trimLeftSlash(head) + "/" + trimLeftSlash(tail)
}
