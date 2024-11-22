package pool

import (
	"context"
	"fmt"
	"sync"
)

type Worker struct {
	p        *Pool
	workChan chan interface{}
	once     sync.Once
	isClosed bool
}

// newWorker new a worker to do the tasks
func newWorker(p *Pool) *Worker {
	return &Worker{
		p:        p,
		workChan: make(chan interface{}, 1),
		once:     sync.Once{},
	}
}

// close the work channel
func (that *Worker) close() {
	that.once.Do(func() {
		that.isClosed = true
		close(that.workChan)
	})
}

// send an arg to the work channel
func (that *Worker) send(arg interface{}) {
	if that.isClosed {
		return
	}
	that.workChan <- arg
}

// Do start to do the task
func (that *Worker) do(ctx context.Context) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				that.p.setError(fmt.Errorf("panic recover err: %v", err))
				that.p.workWg.Done()
			}
		}()
		for {
			select {
			case arg, ok := <-that.workChan:
				if !ok {
					return
				}
				err := that.p.handlerFunc(ctx, arg)
				if err != nil {
					that.p.setError(err)
				}
				that.p.putWorker(that)
				that.p.workWg.Done()
			}
		}
	}()
}
