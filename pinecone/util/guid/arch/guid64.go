package arch

import (
	"encoding/binary"
	"fmt"
)

const (
	SignBits      = 1
	TimestampBits = 29
	WorkerIdBits  = 21
	SequenceBits  = 13
	TotalBits     = 1 << 6 // 64
	Sizeof        = 8
)

type GUID64 struct {
	value int64
}

var _ GUID = GUID64{}

func NewGUID64(value int64) GUID64 {
	return GUID64{value: value}
}

func ParseGUID64(hexID64 string) (GUID64, error) {
	var deltaSeconds, workerId, sequence int64
	n, err := fmt.Sscanf(hexID64, "%x-%x-%x", &deltaSeconds, &workerId, &sequence)
	if err != nil || n != 3 {
		return GUID64{}, fmt.Errorf("invalid GUID64 format: %s", hexID64)
	}

	value := (deltaSeconds << (WorkerIdBits + SequenceBits)) | (workerId << SequenceBits) | sequence
	return GUID64{value: value}, nil
}

func (g GUID64) Int64() int64 {
	return g.value
}

func (g GUID64) Int() int {
	return int(g.value)
}

func (g GUID64) Sizeof() int {
	return Sizeof
}

func (g GUID64) Bitsof() int {
	return g.Sizeof() * 8
}

func (g GUID64) Bytes() []byte {
	return g.BytesLE()
}

func (g GUID64) BytesLE() []byte {
	buf := make([]byte, Sizeof)
	binary.LittleEndian.PutUint64(buf, uint64(g.value))
	return buf
}

func (g GUID64) BytesBE() []byte {
	buf := make([]byte, Sizeof)
	binary.BigEndian.PutUint64(buf, uint64(g.value))
	return buf
}

func (g GUID64) Compare(that Identification) int {
	if that == nil {
		return 1
	}
	thatGUID64, ok := that.(GUID64)
	if !ok {
		return compareString(g.String(), that.String())
	}
	if g.value < thatGUID64.value {
		return -1
	}
	if g.value > thatGUID64.value {
		return 1
	}
	return 0
}

func (g GUID64) HashCode64() uint64 {
	return uint64(g.value)
}

func (g GUID64) Sequence() int64 {
	return int64(uint64(g.value<<(TotalBits-SequenceBits)) >> (TotalBits - SequenceBits))
}

func (g GUID64) WorkerId() int64 {
	return int64(uint64(g.value<<(TimestampBits+SignBits)) >> (TotalBits - WorkerIdBits))
}

func (g GUID64) DeltaSeconds() int64 {
	return int64(uint64(g.value) >> (WorkerIdBits + SequenceBits))
}

func (g GUID64) String() string {
	return fmt.Sprintf("%07x-%06x-%04x", g.DeltaSeconds(), g.WorkerId(), g.Sequence())
}

func (g GUID64) Equal(other GUID64) bool {
	return g.value == other.value
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
