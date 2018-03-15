package simplesync

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Broadcaster is an extremely safe many-to-many lock to be used in
// place of a doneChan or sync.WaitGroup because:
//    1) Closing a doneChan more than once causes a panic
// 	  2) Calling Wait() on sync.WaitGroup more than once causes a panic
//    3) Calling Add() on sync.WaitGroup during a Wait() causes a panic
//    4) Calling Done() on sync.WaitGroup before the first Add() causes a panic
type Broadcaster struct {
	noCopy    noCopy
	startOnce *sync.Once
	startChan chan struct{}
	received  int32
	doneOnce  *sync.Once
	doneChan  chan struct{}
}

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
		startOnce: &sync.Once{},
		startChan: make(chan struct{}, receiverCount),
		received:  0,
		doneOnce:  &sync.Once{},
		doneChan:  make(chan struct{}, 1),
	}
}

// Send unblocks the goroutines and waits for their completion
func (b *Broadcaster) Send() int {
	b.startOnce.Do(func() {
		b.doneOnce = &sync.Once{}
		for i := 0; i < cap(b.startChan); i++ {
			b.startChan <- struct{}{}
		}
		<-b.doneChan
		b.doneChan = make(chan struct{}, 1)
	})
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

// WaitWithTimeout returns an error if the timeout expires
func (b *Broadcaster) WaitWithTimeout(timeout time.Duration) (err error) {
	doneChan := make(chan struct{}, 1)
	go func() {
		b.Receive()
		close(doneChan)
	}()
	select {
	case <-doneChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %v", timeout)
	}
}

// WaitWithDoneChan hands the calling function a doneChan to receive from
func (b *Broadcaster) WaitWithDoneChan() (doneChan chan struct{}) {
	doneChan = make(chan struct{}, 1)
	go func() {
		b.Receive()
		close(doneChan)
	}()
	return doneChan
}
