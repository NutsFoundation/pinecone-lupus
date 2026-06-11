package core

import (
	"io"

	clientarch "github.com/NutsFoundation/pinecone-lupus/pinecone/redstone/shuttle/client/arch"
)

type ObjectStream struct {
	reader   io.ReadCloser
	metadata clientarch.ObjectMetadata
}

var _ clientarch.ObjectStream = (*ObjectStream)(nil)

func NewObjectStream(reader io.ReadCloser, metadata clientarch.ObjectMetadata) *ObjectStream {
	return &ObjectStream{
		reader:   reader,
		metadata: metadata,
	}
}

func (s *ObjectStream) Read(p []byte) (int, error) {
	if s == nil || s.reader == nil {
		return 0, io.EOF
	}
	return s.reader.Read(p)
}

func (s *ObjectStream) Close() error {
	if s == nil || s.reader == nil {
		return nil
	}
	return s.reader.Close()
}

func (s *ObjectStream) Metadata() clientarch.ObjectMetadata {
	if s == nil {
		return clientarch.ObjectMetadata{}
	}
	return s.metadata
}
