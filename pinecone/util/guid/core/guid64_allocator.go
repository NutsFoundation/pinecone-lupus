package core

import (
	"fmt"
	"sync"
	"time"

	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
)

const (
	defaultTimeBits   = 28
	defaultWorkerBits = 22
	defaultSeqBits    = 13
	defaultEpochStr   = "2026-07-01"
)

type DefaultGuid64Allocator struct {
	timeBits   int
	workerBits int
	seqBits    int

	epochSeconds int64
	bitsAlloc    *BitsAllocator
	workerId     int64

	mu         sync.Mutex
	sequence   int64
	lastSecond int64
}

var _ guidarch.GuidAllocator64 = (*DefaultGuid64Allocator)(nil)

type Guid64AllocatorOption func(*DefaultGuid64Allocator)

func WithTimeBits(bits int) Guid64AllocatorOption {
	return func(a *DefaultGuid64Allocator) {
		if bits > 0 {
			a.timeBits = bits
		}
	}
}

func WithWorkerBits(bits int) Guid64AllocatorOption {
	return func(a *DefaultGuid64Allocator) {
		if bits > 0 {
			a.workerBits = bits
		}
	}
}

func WithSeqBits(bits int) Guid64AllocatorOption {
	return func(a *DefaultGuid64Allocator) {
		if bits > 0 {
			a.seqBits = bits
		}
	}
}

func WithEpoch(epochStr string) Guid64AllocatorOption {
	return func(a *DefaultGuid64Allocator) {
		if epochStr != "" {
			parsed, err := time.Parse("2006-01-02", epochStr)
			if err == nil {
				a.epochSeconds = parsed.Unix()
			}
		}
	}
}

func NewDefaultGuid64Allocator(workerIdAssigner guidarch.WorkerIdAssigner, opts ...Guid64AllocatorOption) *DefaultGuid64Allocator {
	epochTime, _ := time.Parse("2006-01-02", defaultEpochStr)

	alloc := &DefaultGuid64Allocator{
		timeBits:     defaultTimeBits,
		workerBits:   defaultWorkerBits,
		seqBits:      defaultSeqBits,
		epochSeconds: epochTime.Unix(),
		lastSecond:   -1,
	}

	for _, opt := range opts {
		opt(alloc)
	}

	alloc.bitsAlloc = NewBitsAllocator(alloc.timeBits, alloc.workerBits, alloc.seqBits)

	alloc.workerId = workerIdAssigner.AssignWorkerId()
	if alloc.workerId > alloc.bitsAlloc.MaxWorkerId() {
		panic(fmt.Sprintf("worker id %d exceeds the max %d", alloc.workerId, alloc.bitsAlloc.MaxWorkerId()))
	}

	return alloc
}

func (a *DefaultGuid64Allocator) NextGUID64() guidarch.GUID64 {
	return guidarch.NewGUID64(a.NextGUIDi64())
}

func (a *DefaultGuid64Allocator) NextGUIDi64() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nextId()
}

func (a *DefaultGuid64Allocator) Explain(guid64 int64) string {
	sequenceBits := a.bitsAlloc.SequenceBits()
	workerIdBits := a.bitsAlloc.WorkerIdBits()
	timestampBits := a.bitsAlloc.TimestampBits()
	signBits := a.bitsAlloc.SignBits()

	sequence := int64(uint64(guid64<<(totalBits-sequenceBits)) >> (totalBits - sequenceBits))
	workerId := int64(uint64(guid64<<(timestampBits+signBits)) >> (totalBits - workerIdBits))
	deltaSeconds := int64(uint64(guid64) >> (workerIdBits + sequenceBits))

	thatTime := time.Unix(a.epochSeconds+deltaSeconds, 0)
	return fmt.Sprintf(
		`{"UID":"%d","timestamp":"%s","workerId":"%d","sequence":"%d"}`,
		guid64, thatTime.Format("2006-01-02 15:04:05"), workerId, sequence,
	)
}

func (a *DefaultGuid64Allocator) nextId() int64 {
	currentSecond := a.getCurrentSecond()

	if currentSecond < a.lastSecond {
		refusedSeconds := a.lastSecond - currentSecond
		panic(fmt.Sprintf("clock moved backwards, refusing for %d seconds", refusedSeconds))
	}

	if currentSecond == a.lastSecond {
		a.sequence = (a.sequence + 1) & a.bitsAlloc.MaxSequence()
		if a.sequence == 0 {
			currentSecond = a.getNextSecond(a.lastSecond)
		}
	} else {
		a.sequence = 0
	}

	a.lastSecond = currentSecond

	return a.bitsAlloc.Allocate(currentSecond-a.epochSeconds, a.workerId, a.sequence)
}

func (a *DefaultGuid64Allocator) getNextSecond(lastTimestamp int64) int64 {
	timestamp := a.getCurrentSecond()
	for timestamp <= lastTimestamp {
		timestamp = a.getCurrentSecond()
	}
	return timestamp
}

func (a *DefaultGuid64Allocator) getCurrentSecond() int64 {
	currentSecond := time.Now().Unix()
	if currentSecond-a.epochSeconds > a.bitsAlloc.MaxDeltaSeconds() {
		panic(fmt.Sprintf("timestamp bits is exhausted, refusing UID generate, now: %d", currentSecond))
	}
	return currentSecond
}
