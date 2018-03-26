package simplesync

import "sync"

// Broadcaster is a flexible many-to-many lock to be used in place of a waitGroup
type Broadcaster struct {
	noCopy    noCopy
	sending   *sync.Mutex
	startChan chan struct{}
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
		startChan: make(chan struct{}, receiverCount),
	}
}

// Send unblocks the goroutines
func (b *Broadcaster) Send() int {
	b.sending.Lock()
	for i := 0; i < cap(b.startChan); i++ {
		b.startChan <- struct{}{}
	}
	b.sending.Unlock()
	return cap(b.startChan)
}

// Receive is called by the goroutines waiting on the Broadcast
func (b *Broadcaster) Receive() {
	<-b.startChan
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
