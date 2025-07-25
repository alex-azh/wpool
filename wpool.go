package wpool

import (
	"context"
)

// WPool[T any] calls the do function in parallel mode until the first error is received.
// After that, only one goroutine will try to do the do until the error is fixed.
// Next, parallel handlers will be started again.
type WPool[T any] struct {
	parallelFactor uint
	tasks          chan T
	do             func(T) error
	timeout        func()
	ctx            context.Context
}

func New[T any](ctx context.Context, parallelFactor uint, do func(T) error, timeout func()) WPool[T] {
	return WPool[T]{parallelFactor, make(chan T, parallelFactor), do, timeout, ctx}
}

// Push item to tasks queue.
func (w *WPool[T]) Push(item T) {
	w.tasks <- item
}

func (w *WPool[T]) Start() {
	// this is ctx will be closed when 1 process exit with error
	ctx, cancel := context.WithCancel(context.Background())

	// todo: can start processing tasks as needed so that they don't just hang in the background.
	w.startParallelWorkers(ctx, cancel)

	// resolver trying invoke do() without error with timeout
	w.startErrResolver(ctx)
}

func (w *WPool[T]) startParallelWorkers(ctx context.Context, cancel context.CancelFunc) {
	for range w.parallelFactor {
		go w.process(ctx, cancel)
	}
}

// startErrResolver trying invoke do() without error with timeout
func (w *WPool[T]) startErrResolver(ctx context.Context) {
	go func() {
		select {
		case <-w.ctx.Done(): // pool must be closed
			return
		case <-ctx.Done(): // some process said it ended with an error
		}

		// get task
		v, ok := <-w.tasks
		_ = ok

		// try do (non parallel, only this doing)
		for {
			w.timeout()
			if w.ctx.Err() != nil {
				return
			}
			if err := w.do(v); err == nil {
				break
			}
		}

		// parallel workers start
		w.Start()
	}()
}

// process take task and doing do() when ctx's not closeds.
func (w *WPool[T]) process(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		// process exit
		case <-ctx.Done():
			return
		// worker pool exit
		case <-w.ctx.Done():
			return
		// read task
		case v, ok := <-w.tasks:
			_ = ok // channel can't be closed
			if err := w.do(v); err != nil {
				w.tasks <- v
				cancel()
				return
			}
		}
	}
}
