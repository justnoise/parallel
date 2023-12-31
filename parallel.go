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
	producer      Producer
	workers       []*Worker
	workQueue     WorkQueue
	resultHandler ResultHandler
}

func NewParallelRunner(producer Producer, executors []Executor, resultHandler ResultHandler, workQueue WorkQueue) *ParallelRunner {
	workers := make([]*Worker, len(executors))
	for i, executor := range executors {
		workers[i] = NewWorker(executor, workQueue, resultHandler, i+1)
	}
	return &ParallelRunner{
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
		// Producer is done. Wait until the work queue is empty before telling the workers to stop.
		keepWaiting := true
		for r.workQueue.Len() > 0 && keepWaiting {
			select {
			case <-ctx.Done():
				log.Println("Context done, not waiting for work queue to be empty")
				keepWaiting = false
			default:
				log.Println("Waiting for workers to finish")
				time.Sleep(1 * time.Second)
			}
		}
	}
	r.workQueue.Stop()
	for _, w := range r.workers {
		w.Stop()
	}
	wg.Wait()
	return err
}
