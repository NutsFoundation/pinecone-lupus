package core

import (
	"crypto/rand"
	"encoding/binary"

	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
)

const defaultWorkerIdBound = 1000

type RandomWorkerIdAssigner struct {
	bound int64
}

var _ guidarch.WorkerIdAssigner = (*RandomWorkerIdAssigner)(nil)

func NewRandomWorkerIdAssigner() *RandomWorkerIdAssigner {
	return &RandomWorkerIdAssigner{bound: defaultWorkerIdBound}
}

func NewRandomWorkerIdAssignerWithBound(bound int64) *RandomWorkerIdAssigner {
	if bound <= 0 {
		bound = defaultWorkerIdBound
	}
	return &RandomWorkerIdAssigner{bound: bound}
}

func (a *RandomWorkerIdAssigner) AssignWorkerId() int64 {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	v := int64(binary.LittleEndian.Uint64(buf[:]))
	if v < 0 {
		v = -v
	}
	return v % a.bound
}
