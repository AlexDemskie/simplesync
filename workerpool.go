package simplesync

// WorkerPool allows parallel execution of an arbitrary func()
type WorkerPool struct {
	work chan func(thread int)
	syn  chan struct{}
	ack  chan struct{}
}

// NewWorkerPool returns a WorkerPool object that facilitates
// parallel processing by calling the Execute() func
func NewWorkerPool(count int) (pool *WorkerPool) {
	if count < 1 {
		return nil
	}
	pool = &WorkerPool{
		work: make(chan func(thread int), count),
		syn:  make(chan struct{}, 1),
		ack:  make(chan struct{}, 1),
	}
	for i := 0; i < count; i++ {
		go func(thread int) {
			for w := range pool.work {
				w(thread)
				pool.syn <- struct{}{}
				<-pool.ack
			}
		}(i)
	}
	return pool
}

// Execute will pass the func() argument to each goroutine whilst
// signaling them to start executing and then block until completion
func (pool *WorkerPool) Execute(someWork func(thread int)) {
	count := cap(pool.work)
	for i := 0; i < count; i++ {
		pool.work <- someWork
	}
	for i := 0; i < count; i++ {
		<-pool.syn
		pool.ack <- struct{}{}
	}
}

// Delete cancels the goroutines
func (pool *WorkerPool) Delete() {
	select {
	case <-pool.work:
		return
	default:
		close(pool.work)
	}
}
