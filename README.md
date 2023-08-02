# Go Generic Worker Pool
[![Build Status](https://github.com/wimspaargaren/workers/workflows/ci/badge.svg)](https://github.com/wimspaargaren/workers/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/wimspaargaren/workers.svg)](https://pkg.go.dev/github.com/wimspaargaren/workers)

This repository provides a generic, easy to use worker pool.

Initialise a buffered or unbuffered pool with a processing function defined as func(U) (T, error). The result types will be inferred from the compiler, from the provided processing function.

The amount of workers and buffer size can be configured.

In case the provided context gets canceled, the workers will stop processing and jobs can no longer be added to the pool. The `AwaitResults` function will return an error `ErrContextCanceled`.

# Example usage

```GOLANG
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/wimspaargaren/workers"
)

// an example processor
type myRandomProcessor struct {
	rand *rand.Rand
}

func (p *myRandomProcessor) process(i int) (string, error) {
	duration := time.Millisecond * time.Duration(p.rand.Intn(100))
	time.Sleep(duration)
	return fmt.Sprintf("processed %d in %d milliseconds", i, duration.Milliseconds()), nil
}

func main() {
	randomProcessor := myRandomProcessor{
		rand: rand.New(rand.NewSource(time.Now().Unix())),
	}
    // create a new default buffered pool
	workerPool := workers.NewBufferedPool(context.Background(),
		randomProcessor.process,
	)
	go func() {
        // add jobs to process
		for i := 0; i < 42; i++ {
			err := workerPool.AddJob(i)
			if err != nil {
				// handle me
			}
		}
		// call done, to indicate all jobs have been added to the pool
		workerPool.Done()
	}()

    // await until all jobs are processed
	results, err := workerPool.AwaitResults()
	if err != nil {
		// handle me
	}
	for i, r := range results {
		fmt.Println("i", i, "r", r)
	}
}
```
