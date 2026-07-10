package core

import (
	"errors"

	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	guidcore "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/core"
)

func ResolveClientID(configured *int64, allocator guidarch.GuidAllocator64) (int64, error) {
	if configured != nil {
		if *configured > 0 {
			return *configured, nil
		}
		if *configured < -1 {
			return 0, errors.New("clientId must be positive, null, 0, or -1")
		}
	}
	if allocator == nil {
		allocator = guidcore.NewDefaultGuid64Allocator(guidcore.NewRandomWorkerIdAssigner())
	}
	return allocator.NextGUIDi64(), nil
}
