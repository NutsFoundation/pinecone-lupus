package core

import "fmt"

const totalBits = 1 << 6 // 64

type BitsAllocator struct {
	signBits      int
	timestampBits int
	workerIdBits  int
	sequenceBits  int

	maxDeltaSeconds int64
	maxWorkerId     int64
	maxSequence     int64

	timestampShift int
	workerIdShift  int
}

func NewBitsAllocator(timestampBits, workerIdBits, sequenceBits int) *BitsAllocator {
	allocateTotalBits := 1 + timestampBits + workerIdBits + sequenceBits
	if allocateTotalBits != totalBits {
		panic(fmt.Sprintf("allocate not enough 64 bits, got %d", allocateTotalBits))
	}

	return &BitsAllocator{
		signBits:        1,
		timestampBits:   timestampBits,
		workerIdBits:    workerIdBits,
		sequenceBits:    sequenceBits,
		maxDeltaSeconds: ^(-1 << timestampBits),
		maxWorkerId:     ^(-1 << workerIdBits),
		maxSequence:     ^(-1 << sequenceBits),
		timestampShift:  workerIdBits + sequenceBits,
		workerIdShift:   sequenceBits,
	}
}

func (b *BitsAllocator) Allocate(deltaSeconds, workerId, sequence int64) int64 {
	return (deltaSeconds << b.timestampShift) | (workerId << b.workerIdShift) | sequence
}

func (b *BitsAllocator) SignBits() int          { return b.signBits }
func (b *BitsAllocator) TimestampBits() int     { return b.timestampBits }
func (b *BitsAllocator) WorkerIdBits() int      { return b.workerIdBits }
func (b *BitsAllocator) SequenceBits() int      { return b.sequenceBits }
func (b *BitsAllocator) MaxDeltaSeconds() int64 { return b.maxDeltaSeconds }
func (b *BitsAllocator) MaxWorkerId() int64     { return b.maxWorkerId }
func (b *BitsAllocator) MaxSequence() int64     { return b.maxSequence }
func (b *BitsAllocator) TimestampShift() int    { return b.timestampShift }
func (b *BitsAllocator) WorkerIdShift() int     { return b.workerIdShift }
