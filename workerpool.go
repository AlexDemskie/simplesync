package simplesync

import "sync"

// WorkerPool allows parallel execution of an arbitrary func()
type WorkerPool struct {
	noCopy     noCopy
	bcastStart *Broadcaster
	bcastEnd   *Broadcaster
	singleExec sync.Mutex
	workFunc   func(thread int)
}

// NewWorkerPool returns a WorkerPool object that facilitates
// parallel processing by calling the Execute() func
func NewWorkerPool(numWorkers int) (pool *WorkerPool) {
	if numWorkers < 1 {
		return nil
	}
	pool = &WorkerPool{
		bcastStart: NewBroadcaster(numWorkers),
		bcastEnd:   NewBroadcaster(1),
		singleExec: sync.Mutex{},
		workFunc:   func(int) {},
	}
	// spawning goroutines to execute work
	go pool.bcastStart.Send()
	for i := 0; i < numWorkers; i++ {
		go func(thread int) {
			for {
				pool.bcastStart.Receive()
				pool.workFunc(thread)
				pool.bcastEnd.Send()
			}
		}(i)
	}
	for i := 0; i < numWorkers; i++ {
		pool.bcastEnd.Receive()
	}
	return pool
}

// Execute will pass the func() argument to each goroutine whilst
// signaling them to start executing and then block until completion
func (pool *WorkerPool) Execute(someWork func(thread int)) {
	pool.singleExec.Lock()
	// pass the function to the goroutines
	pool.workFunc = someWork
	// signal to the worker goroutines to start executing
	// then wait for completion confirmation from each worker
	pool.bcastStart.Send()
	for i := 0; i < pool.bcastStart.Len(); i++ {
		pool.bcastEnd.Receive()
	}
	pool.singleExec.Unlock()
}
