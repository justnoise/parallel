package parallel

import (
	"context"
	"log"
	"sync"
	"time"
)

type Executor interface {
	Do(context.Context, interface{}) (interface{}, error)
}

type Producer interface {
	Produce(context.Context, WorkQueue) error
}

type ResultHandler interface {
	Handle(context.Context, interface{}, error)
}

type ParallelRunner struct {
	numWorkers    int
	producer      Producer
	workers       []*Worker
	workQueue     WorkQueue
	resultHandler ResultHandler
}

func NewParallelRunner(numWorkers int, producer Producer, executor Executor, resultHandler ResultHandler, workQueue WorkQueue) *ParallelRunner {
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(executor, workQueue, resultHandler, i+1)
	}
	return &ParallelRunner{
		numWorkers:    numWorkers,
		producer:      producer,
		workers:       workers,
		workQueue:     workQueue,
		resultHandler: resultHandler,
	}
}

func (r *ParallelRunner) Run(inputCtx context.Context) error {
	ctx, cancel := context.WithCancel(inputCtx)
	defer cancel()
	var wg sync.WaitGroup
	for _, w := range r.workers {
		go w.Run(ctx, &wg)
	}
	err := r.producer.Produce(ctx, r.workQueue)
	if err != nil {
		log.Println("Producer error: ", err)
	} else {
		// Producer is done. Poll until all workers finish
		keepWaiting := true
		for !r.workQueue.Empty() && keepWaiting {
			select {
			case <-ctx.Done():
				log.Println("Context done, not waiting for workers")
				keepWaiting = false
			default:
				log.Println("Waiting for workers to finish")
				time.Sleep(1 * time.Second)
			}
		}
	}
	cancel()
	wg.Wait()
	return err
}
