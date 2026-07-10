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

type correlationWaiter struct {
	mu      sync.Mutex
	waiters map[string]chan correlationResult
}

func newCorrelationWaiter() *correlationWaiter {
	return &correlationWaiter{waiters: map[string]chan correlationResult{}}
}

func (w *correlationWaiter) prepare(guid string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.waiters[guid] = make(chan correlationResult, 1)
}

func (w *correlationWaiter) await(ctx context.Context, guid string) (*lifecyclepb.ServiceControlFrame, error) {
	w.mu.Lock()
	ch := w.waiters[guid]
	w.mu.Unlock()
	if ch == nil {
		return nil, nil
	}
	defer w.remove(guid)
	select {
	case result := <-ch:
		return result.frame, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (w *correlationWaiter) complete(guid string, frame *lifecyclepb.ServiceControlFrame) {
	w.mu.Lock()
	ch := w.waiters[guid]
	w.mu.Unlock()
	if ch != nil {
		ch <- correlationResult{frame: frame}
	}
}

func (w *correlationWaiter) completeExceptionally(guid string, err error) {
	w.mu.Lock()
	ch := w.waiters[guid]
	w.mu.Unlock()
	if ch != nil {
		ch <- correlationResult{err: err}
	}
}

func (w *correlationWaiter) completeAllExceptionally(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for guid, ch := range w.waiters {
		ch <- correlationResult{err: err}
		delete(w.waiters, guid)
	}
}

func (w *correlationWaiter) remove(guid string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.waiters, guid)
}
