package core

import (
	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	"github.com/google/uuid"
)

type UUIDV7GuidAllocator struct{}

var _ guidarch.GuidAllocator = (*UUIDV7GuidAllocator)(nil)

func NewUUIDV7GuidAllocator() *UUIDV7GuidAllocator {
	return &UUIDV7GuidAllocator{}
}

func (a *UUIDV7GuidAllocator) NextGUID() guidarch.GUID {
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	return NewUUIDGUID(id)
}

func (a *UUIDV7GuidAllocator) Parse(raw string) (guidarch.GUID, error) {
	return NewUUIDGUIDFromString(raw)
}
