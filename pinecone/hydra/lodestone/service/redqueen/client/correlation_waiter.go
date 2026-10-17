package client

import (
	"context"
	"sync"

	lifecyclepb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/lifecycle"
)

type correlationResult struct {
	frame *lifecyclepb.ServiceControlFrame
	err   error
}

type pendingCorrelation struct {
	guid   string
	done   chan struct{}
	once   sync.Once
	result correlationResult
}

func (p *pendingCorrelation) complete(result correlationResult) {
	p.once.Do(func() {
		p.result = result
		close(p.done)
	})
}

type correlationWaiter struct {
	mu          sync.Mutex
	waiters     map[string]*pendingCorrelation
	terminalErr error
}

func newCorrelationWaiter() *correlationWaiter {
	return &correlationWaiter{waiters: map[string]*pendingCorrelation{}}
}

func (w *correlationWaiter) prepare(guid string) *pendingCorrelation {
	w.mu.Lock()
	defer w.mu.Unlock()
	pending := &pendingCorrelation{guid: guid, done: make(chan struct{})}
	if w.terminalErr != nil {
		pending.complete(correlationResult{err: w.terminalErr})
		return pending
	}
	w.waiters[guid] = pending
	return pending
}

func (w *correlationWaiter) await(ctx context.Context, pending *pendingCorrelation) (*lifecyclepb.ServiceControlFrame, error) {
	select {
	case <-pending.done:
		return pending.result.frame, pending.result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (w *correlationWaiter) complete(guid string, frame *lifecyclepb.ServiceControlFrame) {
	w.mu.Lock()
	pending := w.waiters[guid]
	w.mu.Unlock()
	if pending != nil {
		pending.complete(correlationResult{frame: frame})
	}
}

func (w *correlationWaiter) completeExceptionally(guid string, err error) {
	w.mu.Lock()
	pending := w.waiters[guid]
	w.mu.Unlock()
	if pending != nil {
		pending.complete(correlationResult{err: err})
	}
}

func (w *correlationWaiter) completeAllExceptionally(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.terminalErr == nil {
		w.terminalErr = err
	}
	for guid, pending := range w.waiters {
		pending.complete(correlationResult{err: err})
		delete(w.waiters, guid)
	}
}

func (w *correlationWaiter) remove(pending *pendingCorrelation) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.waiters[pending.guid] == pending {
		delete(w.waiters, pending.guid)
	}
}
