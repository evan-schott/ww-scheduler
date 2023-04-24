package syncutil

import "sync/atomic"

type Barrier uint32

func (b *Barrier) Signal(finalize func()) (ready bool) {
	if ready = atomic.AddUint32((*uint32)(b), ^uint32(0)) == 0; ready {
		finalize()
	}

	return
}

type BarrierChan struct {
	b  *Barrier
	cq chan struct{}
}

func NewBarrierChan(n uint32) BarrierChan {
	return BarrierChan{
		b:  (*Barrier)(&n),
		cq: make(chan struct{}),
	}
}

func (b BarrierChan) Done() <-chan struct{} { return b.cq }

func (b BarrierChan) Signal(finalize func()) (ready bool) {
	return b.b.Signal(func() {
		defer close(b.cq)
		finalize()
	})
}

func (b BarrierChan) SignalAndWait(finalize func()) {
	b.Signal(finalize)
	<-b.cq
}
