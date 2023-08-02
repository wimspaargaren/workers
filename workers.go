// Package workers provides a generic worker pool.
package workers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	defaultWorkerSize = 10
	defaultBufferSize = 100
)

// Error definition
var (
	ErrResultsAlreadyRead = fmt.Errorf("results have already been read")
	ErrContextCanceled    = fmt.Errorf("context was canceled")
)

// Pool a worker pool to process a list of jobs by a given amount of workers
// which process the added jobs concurrently.
type Pool[T any, U any] interface {
	// AddJob adds a job of type T to the Pool. If Done() was already called
	// on the pool, the AddJob function will return an error.
	AddJob(T) error
	// Done indicates that all jobs are added tot he pool. After this method is called,
	// we can no longer add jobs to the pool, until we reset the pool.
	Done()
	// AwaitResults waits until the workers have processed all jobs. Important: make
	// sure to call the Done() function in order to close the job channel and let the workers
	// stop once all jobs are processed. Calling the Done() function can be done concurrently.
	AwaitResults() ([]U, error)
	// In case you want to handle the results yourself, you can retrieve the result and error
	// channels from the pool. Keep in mind that these channels will not close until you call
	// the Done() function.
	ResultChannels() (<-chan U, <-chan error)
}

// OptsFunc options function.
type OptsFunc func(*Options)

// Options set the worker pool options.
type Options struct {
	// Defaults to 10 workers
	Workers int
	// Defaults to a buffer size of 100
	BufferSize int
}

// WithBufferSize adds a specified buffer size to the options.
func WithBufferSize(bufferSize int) OptsFunc {
	return func(o *Options) {
		o.BufferSize = bufferSize
	}
}

// WithWorkers adds a specified amount of workers to the options.
func WithWorkers(workers int) OptsFunc {
	return func(o *Options) {
		o.Workers = workers
	}
}

func defaultOpts() *Options {
	return &Options{
		Workers:    defaultWorkerSize,
		BufferSize: defaultBufferSize,
	}
}

type bufferedPool[T any, U any] struct {
	pool[T, U]
}

// NewBufferedPool creates a new pool with buffered channels.
func NewBufferedPool[T any, U any](ctx context.Context, processFunc func(T) (U, error), opts ...OptsFunc) Pool[T, U] {
	bufferedPool := &bufferedPool[T, U]{
		newPool[T, U](ctx, processFunc, opts...),
	}
	bufferedPool.reset()

	return bufferedPool
}

type unBufferedPool[T any, U any] struct {
	pool[T, U]
}

// NewUnBufferedPool creates a new pool with unbuffered channels.
func NewUnBufferedPool[T any, U any](ctx context.Context, processFunc func(T) (U, error), opts ...OptsFunc) Pool[T, U] {
	unBufferedPool := &unBufferedPool[T, U]{
		newPool[T, U](ctx, processFunc, opts...),
	}
	unBufferedPool.ResetUnBuffered()

	return unBufferedPool
}

func newPool[T any, U any](ctx context.Context, processFunc func(T) (U, error), opts ...OptsFunc) pool[T, U] {
	options := defaultOpts()

	for _, f := range opts {
		f(options)
	}

	return pool[T, U]{
		options:     options,
		processFunc: processFunc,
		wg:          &sync.WaitGroup{},
		ctx:         ctx,
	}
}

type pool[T any, U any] struct {
	options *Options

	wg            *sync.WaitGroup
	jobChannel    chan T
	resultChannel chan U
	errChannel    chan error

	processFunc func(T) (U, error)
	hasFinished atomic.Bool
	ctx         context.Context
}

// AddJob adds a job to the pool; Returns an error
// if the result channel has been closed already.
func (w *pool[T, U]) AddJob(job T) error {
	if w.hasFinished.Load() {
		return ErrResultsAlreadyRead
	}
	w.jobChannel <- job
	return nil
}

// ResultChannels returns the result channels as read only.
func (w *pool[T, U]) ResultChannels() (<-chan U, <-chan error) {
	return w.resultChannel, w.errChannel
}

// AwaitResults awaits until all jobs have been processed. Make sure to
// call the done function in order for the result and error channel to be
// closed.
func (w *pool[T, U]) AwaitResults() ([]U, error) {
	results := []U{}
	errorList := []error{}

	for {
		select {
		case v, ok := <-w.errChannel:
			if !ok {
				w.errChannel = nil
			} else {
				errorList = append(errorList, v)
			}
		case v, ok := <-w.resultChannel:
			if !ok {
				w.resultChannel = nil
			} else {
				results = append(results, v)
			}
		case <-w.ctx.Done():
			return results, fmt.Errorf("%w; %w", ErrContextCanceled, errors.Join(errorList...))
		}
		if w.errChannel == nil && w.resultChannel == nil {
			break
		}
	}

	if len(errorList) > 0 {
		return results, errors.Join(errorList...)
	}
	return results, nil
}

// Done indicate that all jobs have been published and
// close the job channel.
func (w *pool[T, U]) Done() {
	if !w.hasFinished.Load() {
		w.setHasFinished(true)
		close(w.jobChannel)
	}
}

// reset reset the channels of the
func (w *pool[T, U]) reset() {
	w.initBufferedChannels()
	w.resetWorkersAndAwaitProcessingDone()
}

func (w *pool[T, U]) ResetUnBuffered() {
	w.initUnBufferedChannels()
	w.reset()
}

func (w *pool[T, U]) resetWorkersAndAwaitProcessingDone() {
	w.setupWorkers()
	w.setHasFinished(false)
	go w.awaitProcessingDone()
}

func (w *pool[T, U]) initBufferedChannels() {
	w.jobChannel = make(chan T, w.options.BufferSize)
	w.resultChannel = make(chan U, w.options.BufferSize)
	w.errChannel = make(chan error)
}

func (w *pool[T, U]) initUnBufferedChannels() {
	w.jobChannel = make(chan T)
	w.resultChannel = make(chan U)
	w.errChannel = make(chan error)
}

func (w *pool[T, U]) setupWorkers() {
	for i := 0; i < w.options.Workers; i++ {
		w.wg.Add(1)
		go w.worker()
	}
}

func (w *pool[T, U]) worker() {
	defer w.wg.Done()
	for {
		select {
		case job, ok := <-w.jobChannel:
			if !ok {
				return
			}
			res, err := w.processFunc(job)
			if err != nil {
				w.errChannel <- err
				continue
			}
			w.resultChannel <- res
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *pool[T, U]) awaitProcessingDone() {
	w.wg.Wait()
	w.setHasFinished(true)
	close(w.resultChannel)
	close(w.errChannel)
}

func (w *pool[T, U]) setHasFinished(hasFinished bool) {
	w.hasFinished.Store(hasFinished)
}
