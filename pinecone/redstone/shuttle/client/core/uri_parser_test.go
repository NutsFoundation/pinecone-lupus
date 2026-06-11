package core

import (
	"testing"

	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

func TestURIParserObjectWithExplicitEndpoint(t *testing.T) {
	parser := NewURIParser(clientarch.DefaultSystemBucket)
	uri, err := parser.Parse("red://localhost:5477/root@block-striped/avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if uri.Endpoint != "localhost:5477" {
		t.Fatalf("endpoint = %q", uri.Endpoint)
	}
	if uri.Bucket != "root@block-striped" {
		t.Fatalf("bucket = %q", uri.Bucket)
	}
	if uri.Key != "avatar.png" {
		t.Fatalf("key = %q", uri.Key)
	}
	if !uri.ExplicitEndpoint {
		t.Fatalf("explicit endpoint = false")
	}
	if uri.ResourceType != clientarch.ResourceTypeObject {
		t.Fatalf("resource type = %q", uri.ResourceType)
	}
}

func TestURIParserObjectWithDefaultEndpoint(t *testing.T) {
	parser := NewURIParser(clientarch.DefaultSystemBucket)
	uri, err := parser.Parse("red://root@block-striped/demo.txt")
	if err != nil {
		t.Fatal(err)
	}
	if uri.Endpoint != "" {
		t.Fatalf("endpoint = %q", uri.Endpoint)
	}
	if uri.Bucket != "root@block-striped" {
		t.Fatalf("bucket = %q", uri.Bucket)
	}
	if uri.Key != "demo.txt" {
		t.Fatalf("key = %q", uri.Key)
	}
}

func TestURIParserKernelPath(t *testing.T) {
	parser := NewURIParser(clientarch.DefaultSystemBucket)
	uri, err := parser.Parse("red:///proc/1234/status")
	if err != nil {
		t.Fatal(err)
	}
	if uri.Bucket != clientarch.DefaultSystemBucket {
		t.Fatalf("bucket = %q", uri.Bucket)
	}
	if uri.Key != "proc/1234/status" {
		t.Fatalf("key = %q", uri.Key)
	}
	if uri.ResourceType != clientarch.ResourceTypeKernel {
		t.Fatalf("resource type = %q", uri.ResourceType)
	}
}

func TestURIParserKernelBucket(t *testing.T) {
	parser := NewURIParser(clientarch.DefaultSystemBucket)
	uri, err := parser.Parse("s3://__SYSTEM__/proc/1234/status")
	if err != nil {
		t.Fatal(err)
	}
	if uri.Bucket != clientarch.DefaultSystemBucket {
		t.Fatalf("bucket = %q", uri.Bucket)
	}
	if uri.Key != "proc/1234/status" {
		t.Fatalf("key = %q", uri.Key)
	}
	if uri.ResourceType != clientarch.ResourceTypeKernel {
		t.Fatalf("resource type = %q", uri.ResourceType)
	}
}

func TestURIParserUOFSAlias(t *testing.T) {
	parser := NewURIParser(clientarch.DefaultSystemBucket)
	uri, err := parser.Parse("uofs://root@block-striped/demo.txt")
	if err != nil {
		t.Fatal(err)
	}
	if uri.Scheme != "uofs" {
		t.Fatalf("scheme = %q", uri.Scheme)
	}
	if uri.Bucket != "root@block-striped" {
		t.Fatalf("bucket = %q", uri.Bucket)
	}
}
