package pool

import (
	"context"
	"sync"
)

type HandlerFunc func(ctx context.Context, arg interface{}) error

type Option func(p *Pool)

// WithIgnoreErr ignore error
func WithIgnoreErr() Option {
	return func(p *Pool) {
		p.ignoreErr = true
	}
}

// WithSize set the size of the pool
func WithSize(size uint) Option {
	return func(p *Pool) {
		p.capacity = int32(size)
	}
}

type Pool struct {
	handlerFunc HandlerFunc
	capacity    int32
	err         error
	errLock     sync.Mutex
	ignoreErr   bool
	workerChan  chan *Worker
	workWg      sync.WaitGroup
	release     chan struct{}
}

// NewPool new a goroutine pool
func NewPool(ctx context.Context, options ...Option) *Pool {
	// init a pool instance
	instance := &Pool{
		workWg:    sync.WaitGroup{},
		errLock:   sync.Mutex{},
		capacity:  5,
		release:   make(chan struct{}, 1),
		ignoreErr: false,
	}
	for _, option := range options {
		option(instance)
	}
	instance.workerChan = make(chan *Worker, instance.capacity)
	// init workers
	go instance.initWorkers(ctx)
	return instance
}

// initWorkers init workers
func (that *Pool) initWorkers(ctx context.Context) {
	for i := 0; i < int(that.capacity); i++ {
		worker := newWorker(that)
		worker.do(ctx)
		that.putWorker(worker)
	}
}

// getWorker get a available worker to run the tasks.
func (that *Pool) getWorker(ctx context.Context) *Worker {
	var worker *Worker
	for {
		select {
		case <-ctx.Done():
			that.setError(ctx.Err())
			return nil
		case worker = <-that.workerChan:
			return worker
		}
	}
}

// putWorker put a worker back into the pool
func (that *Pool) putWorker(w *Worker) {
	that.workerChan <- w
}

// Submit a task to pool
func (that *Pool) Submit(ctx context.Context, msgs ...interface{}) {
	for _, msg := range msgs {
		if len(that.release) > 0 {
			return
		}
		// get a worker to do the task
		if worker := that.getWorker(ctx); worker != nil {
			that.workWg.Add(1)
			worker.send(msg)
		}
	}
}

// Wait to finish all the task
func (that *Pool) Wait() error {
	if len(that.release) == 0 {
		that.release <- struct{}{}
	}
	that.workWg.Wait()
	return that.err
}

// RegisterHandleFunc register handler function
func (that *Pool) RegisterHandleFunc(f HandlerFunc) {
	that.handlerFunc = f
}

// setError set an error
func (that *Pool) setError(err error) {
	that.errLock.Lock()
	defer that.errLock.Unlock()
	that.err = err
	if !that.ignoreErr {
		that.release <- struct{}{}
	}
	return
}

// Go new a pool to do the tasks
func Go(ctx context.Context, num uint, f HandlerFunc, params ...interface{}) error {
	p := NewPool(ctx, WithSize(num))
	p.RegisterHandleFunc(f)
	p.Submit(ctx, params...)
	return p.Wait()
}
