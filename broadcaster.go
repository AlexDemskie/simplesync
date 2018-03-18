package simplesync

import (
	"sync"
	"sync/atomic"
)

// Broadcaster is a flexible many-to-many lock to be used in place of a waitGroup
type Broadcaster struct {
	noCopy    noCopy
	sending   *sync.Mutex
	startOnce *sync.Once
	startChan chan struct{}
	received  int32
	doneOnce  *sync.Once
	doneChan  chan struct{}
}

// Len gets the number of receivers
func (b *Broadcaster) Len() int { return cap(b.startChan) }

// See https://github.com/golang/go/issues/8005#issuecomment-190753527
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock() {}

// NewBroadcaster returns an empty Broadcaster object
func NewBroadcaster(receiverCount int) (b *Broadcaster) {
	if receiverCount < 1 {
		return nil
	}
	return &Broadcaster{
		sending:   &sync.Mutex{},
		startOnce: &sync.Once{},
		startChan: make(chan struct{}, receiverCount),
		received:  0,
		doneOnce:  &sync.Once{},
		doneChan:  make(chan struct{}, 1),
	}
}

// Send unblocks the goroutines and waits for their completion
func (b *Broadcaster) Send() int {
	b.sending.Lock()
	b.startOnce.Do(func() {
		b.doneOnce = &sync.Once{}
		for i := 0; i < cap(b.startChan); i++ {
			b.startChan <- struct{}{}
		}
		<-b.doneChan
		b.doneChan = make(chan struct{}, 1)
	})
	b.sending.Unlock()
	return cap(b.startChan)
}

// Receive is called by the goroutines waiting on the Broadcast
func (b *Broadcaster) Receive() {
	<-b.startChan
	if atomic.AddInt32(&b.received, 1) >= int32(cap(b.startChan)) {
		b.doneOnce.Do(func() {
			atomic.AddInt32(&b.received, -int32(cap(b.startChan)))
			b.startOnce = &sync.Once{}
			close(b.doneChan)
		})
	}
	return
}

// ReceiveWithDoneChan hands the calling function a doneChan to receive from
func (b *Broadcaster) ReceiveWithDoneChan() (doneChan chan struct{}) {
	doneChan = make(chan struct{}, 1)
	go func() {
		b.Receive()
		close(doneChan)
	}()
	return doneChan
}
