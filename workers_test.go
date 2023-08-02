package workers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ErrTest = fmt.Errorf("aaaah")

func testProcessingFunc(i int) (int, error) {
	return i, nil
}

func TestBufferedWorkerPool(t *testing.T) {
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		testProcessingFunc,
		WithBufferSize(42),
	)
	processIntIntPool(t, bufferedWorkerPool)
}

func TestBufferedWorkerPoolGoroutine(t *testing.T) {
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		testProcessingFunc,
	)
	processIntIntPoolJobAsync(t, bufferedWorkerPool)
}

func TestUnBufferedWorkerPool(t *testing.T) {
	bufferedWorkerPool := NewUnBufferedPool[int, int](
		context.Background(),
		testProcessingFunc,
	)
	processIntIntPoolJobAsync(t, bufferedWorkerPool)
}

func TestCallDoneTwice(t *testing.T) {
	bufferedWorkerPool := NewUnBufferedPool[int, int](
		context.Background(),
		testProcessingFunc,
		WithWorkers(42),
	)
	go func() {
		for i := 0; i < 42; i++ {
			assert.NoError(t, bufferedWorkerPool.AddJob(i))
		}
		bufferedWorkerPool.Done()
		bufferedWorkerPool.Done()
	}()
	res, err := bufferedWorkerPool.AwaitResults()
	assert.NoError(t, err)
	assert.Equal(t, 42, len(res))
}

func TestProcessWithErrors(t *testing.T) {
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		func(i int) (int, error) {
			if i > 20 {
				return 0, ErrTest
			}
			return i, nil
		},
		WithBufferSize(42),
	)
	for i := 0; i < 42; i++ {
		assert.NoError(t, bufferedWorkerPool.AddJob(i))
	}
	bufferedWorkerPool.Done()
	res, err := bufferedWorkerPool.AwaitResults()
	assert.Error(t, err)

	assert.ErrorIs(t, err, ErrTest)
	assert.Equal(t, 21, len(res))
}

func TestCantAddJobToFinishedPool(t *testing.T) {
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		testProcessingFunc,
		WithBufferSize(42),
	)
	bufferedWorkerPool.Done()
	err := bufferedWorkerPool.AddJob(42)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrResultsAlreadyRead)
}

func TestContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctx, c := context.WithTimeout(ctx, time.Millisecond)
	defer c()
	bufferedWorkerPool := NewUnBufferedPool[int, int](
		ctx,
		testProcessingFunc,
	)
	bufferedWorkerPool.AddJob(42)
	res, err := bufferedWorkerPool.AwaitResults()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContextCanceled)
	assert.Equal(t, 1, len(res))
}

func TestChainPools(t *testing.T) {
	ctx := context.Background()
	bufferedWorkerPool1 := NewUnBufferedPool[int, int](
		ctx,
		testProcessingFunc,
	)
	bufferedWorkerPool2 := NewUnBufferedPool[int, int](
		ctx,
		testProcessingFunc,
	)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for i := 0; i < 42; i++ {
			assert.NoError(t, bufferedWorkerPool1.AddJob(i))
		}
		bufferedWorkerPool1.Done()
	}()

	resChan, errChan := bufferedWorkerPool1.ResultChannels()
	go func() {
		defer wg.Done()
		for e := range errChan {
			assert.NoError(t, e)
		}
	}()

	go func() {
		defer wg.Done()
		for r := range resChan {
			assert.NoError(t, bufferedWorkerPool2.AddJob(r))
		}
		bufferedWorkerPool2.Done()
	}()
	wg.Wait()
	res, err := bufferedWorkerPool2.AwaitResults()
	assert.NoError(t, err)
	assert.Equal(t, 42, len(res))
}

func processIntIntPool(t *testing.T, p Pool[int, int]) {
	for i := 0; i < 42; i++ {
		assert.NoError(t, p.AddJob(i))
	}
	p.Done()
	res, err := p.AwaitResults()
	assert.NoError(t, err)
	assert.Equal(t, 42, len(res))
}

func processIntIntPoolJobAsync(t *testing.T, p Pool[int, int]) {
	go func() {
		for i := 0; i < 42; i++ {
			assert.NoError(t, p.AddJob(i))
		}
		p.Done()
	}()
	res, err := p.AwaitResults()
	assert.NoError(t, err)
	assert.Equal(t, 42, len(res))
}
