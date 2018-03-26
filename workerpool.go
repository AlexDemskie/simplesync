package simplesync

import "sync"

// WorkerPool allows parallel execution of an arbitrary func()
type WorkerPool struct {
	noCopy     noCopy
	bcastStart *Broadcaster
	singleExec *sync.Mutex
	workFunc   func(thread int)
	finished   *sync.WaitGroup
	doneChan   chan struct{}
}

// NewWorkerPool returns a WorkerPool object that facilitates
// parallel processing by calling the Execute() func
func NewWorkerPool(numWorkers int) (pool *WorkerPool) {
	if numWorkers < 1 {
		return nil
	}
	pool = &WorkerPool{
		bcastStart: NewBroadcaster(numWorkers),
		singleExec: &sync.Mutex{},
		workFunc:   func(int) {},
		finished:   &sync.WaitGroup{},
		doneChan:   make(chan struct{}, 1),
	}
	pool.finished.Add(numWorkers)
	go pool.bcastStart.Send()
	// spawning goroutines to execute work
	for i := 0; i < numWorkers; i++ {
		go func(thread int) {
			for {
				pool.bcastStart.Receive()
				pool.workFunc(thread)
				pool.finished.Done()
				select {
				case <-pool.doneChan:
					return
				default:
					continue
				}
			}
		}(i)
	}
	pool.finished.Wait()
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
	pool.finished.Add(pool.bcastStart.Len())
	pool.bcastStart.Send()
	pool.finished.Wait()
	pool.singleExec.Unlock()
}

// Delete cancels the goroutines
func (pool *WorkerPool) Delete() {
	pool.singleExec.Lock()
	defer pool.singleExec.Unlock()
	// ensure that we aren't double closing a channel
	select {
	case <-pool.doneChan:
		return
	default:
		close(pool.doneChan)
	}
	// now that the doneChan is closed we will unblock the workers
	// so that they may see the closed doneChan terminate their goroutines
	pool.workFunc = func(int) {}
	pool.finished.Add(pool.bcastStart.Len())
	go pool.bcastStart.Send()
	pool.finished.Wait()
}
