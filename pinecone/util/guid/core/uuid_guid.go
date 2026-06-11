package core

import (
	"encoding/binary"
	"hash/fnv"
	"strings"

	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	"github.com/google/uuid"
)

const uuidGUIDSizeof = 16

type UUIDGUID struct {
	value uuid.UUID
}

var _ guidarch.GUID = UUIDGUID{}

func NewUUIDGUID(value uuid.UUID) UUIDGUID {
	return UUIDGUID{value: value}
}

func NewUUIDGUIDFromString(raw string) (UUIDGUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return UUIDGUID{}, err
	}
	return NewUUIDGUID(id), nil
}

func (g UUIDGUID) UUID() uuid.UUID {
	return g.value
}

func (g UUIDGUID) String() string {
	return strings.ToLower(g.value.String())
}

func (g UUIDGUID) Bytes() []byte {
	return g.BytesBE()
}

func (g UUIDGUID) BytesLE() []byte {
	bytes := g.BytesBE()
	for left, right := 0, len(bytes)-1; left < right; left, right = left+1, right-1 {
		bytes[left], bytes[right] = bytes[right], bytes[left]
	}
	return bytes
}

func (g UUIDGUID) BytesBE() []byte {
	bytes := make([]byte, uuidGUIDSizeof)
	copy(bytes, g.value[:])
	return bytes
}

func (g UUIDGUID) Compare(that guidarch.Identification) int {
	if that == nil {
		return 1
	}
	return compareString(g.String(), that.String())
}

func (g UUIDGUID) Int64() int64 {
	return int64(g.HashCode64())
}

func (g UUIDGUID) Int() int {
	return int(g.Int64())
}

func (g UUIDGUID) Sizeof() int {
	return uuidGUIDSizeof
}

func (g UUIDGUID) Bitsof() int {
	return g.Sizeof() * 8
}

func (g UUIDGUID) HashCode64() uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write(g.value[:])
	return hash.Sum64()
}

func (g UUIDGUID) High64() uint64 {
	return binary.BigEndian.Uint64(g.value[0:8])
}

func (g UUIDGUID) Low64() uint64 {
	return binary.BigEndian.Uint64(g.value[8:16])
}

func compareString(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
