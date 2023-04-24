package syncutil

import (
	"context"
	"sync"
)

// A Breaker calls functions in separate goroutines, returning a nil
// error if one succeeds.  Contrary to 'Any', calls to 'Wait" will
// block until either a function returns a nil error or all inflight
// functions have returned errors.
//
// The zero-value Breaker is ready to use.
type Breaker struct {
	mu      sync.Mutex
	ctr     int
	brk, ok bool

	once, closeOnce sync.Once
	done            chan struct{}

	errOnce sync.Once
	err     error

	cancel context.CancelFunc
}

// BreakerWithContext returns a Breaker and an associated context derived
// from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// succeeds (returns a nil error) or the first time Wait returns, whichever occurs
// first.
func BreakerWithContext(ctx context.Context) (*Breaker, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Breaker{cancel: cancel}, ctx
}

// Go calls the function in a separate goroutine.  If 'Break' has been
// previously called, this becomes a nop.
func (b *Breaker) Go(f func() error) {
	b.once.Do(b.init)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.brk {
		return
	}

	b.ctr++

	go func() {
		defer func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if b.ctr--; b.ctr == 0 && b.brk {
				close(b.done)
			}
		}()

		if err := f(); err != nil {
			b.errOnce.Do(func() { b.err = err })
			return
		}

		b.closeOnce.Do(func() {
			b.ok = true
			close(b.done)
			if b.cancel != nil {
				b.cancel()
			}
		})
	}()
}

// Break renders each subsequent calls to 'Go' a nop.  Aftr 'Break'
// returns, 'Wait' will block until all in-flight functions have
// returned.
func (b *Breaker) Break() {
	b.once.Do(b.init)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.brk = true; b.ctr == 0 {
		b.closeOnce.Do(func() {
			close(b.done)
			if b.cancel != nil {
				b.cancel()
			}
		})
	}
}

// Wait blocks until a function call succeeds, or until 'Break' has
// been called AND all in-flight functions have returned an error.
// In the latter case, it returns the first non-nil error encountered.
func (b *Breaker) Wait() error {
	b.once.Do(b.init)

	<-b.done
	if b.ok {
		return nil
	}

	return b.err
}

func (b *Breaker) init() { b.done = make(chan struct{}) }
