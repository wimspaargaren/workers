package workers

import (
	"context"
	"fmt"
	"time"
)

func ExampleNewBufferedPool() {
	myProcessingFunc := func(i int) (int, error) {
		// Fictional slow processing method
		time.Sleep(time.Millisecond)
		return i, nil
	}
	amountOfJobs := 42
	// Create a new buffered pool. Note that the result
	// types wil be inferred because of the defined
	// process function.
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		myProcessingFunc,
		// Optionally set a given buffer size.
		WithBufferSize(amountOfJobs),
	)
	// Add 42 jobs to the pool
	for i := 0; i < amountOfJobs; i++ {
		// The job type is inferred as int because of our processing function
		// provided in the NewBufferedPool
		bufferedWorkerPool.AddJob(i)
	}
	// Indicate that all jobs are added tot he pool.
	bufferedWorkerPool.Done()
	// Await the processing results.
	res, err := bufferedWorkerPool.AwaitResults()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got: %d results after processing\n", len(res))
	// Output: Got: 42 results after processing
}

func ExampleNewUnBufferedPool() {
	myProcessingFunc := func(i int) (int, error) {
		// Fictional slow processing method
		time.Sleep(time.Millisecond)
		return i, nil
	}
	amountOfJobs := 42
	// Create a new buffered pool. Note that the result
	// types wil be inferred because of the defined
	// process function.
	bufferedWorkerPool := NewBufferedPool(
		context.Background(),
		myProcessingFunc,
		// Optionally set a given buffer size.
		WithBufferSize(amountOfJobs),
	)
	// Since we have an unbuffered pool make sure that jobs are added
	// in a goroutine, to prevent a deadlock.
	go func() {
		// Add 42 jobs to the pool
		for i := 0; i < amountOfJobs; i++ {
			// The job type is inferred as int because of our processing function
			// provided in the NewBufferedPool
			bufferedWorkerPool.AddJob(i)
		}
		// Indicate that all jobs are added tot he pool.
		bufferedWorkerPool.Done()
	}()
	// Await the processing results.
	res, err := bufferedWorkerPool.AwaitResults()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got: %d results after processing\n", len(res))
	// Output: Got: 42 results after processing
}
