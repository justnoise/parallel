package parallel

import (
	"context"
	"log"
	"sync"
)

type Worker struct {
	executor       Executor
	workQueue      WorkQueue
	resultHandler  ResultHandler
	workerQuitChan chan bool
	// used in testing
	running bool
	id      int
}

func NewWorker(executor Executor, workQueue WorkQueue, resultHandler ResultHandler, id int) *Worker {
	return &Worker{
		executor:       executor,
		workQueue:      workQueue,
		resultHandler:  resultHandler,
		workerQuitChan: make(chan bool),
		running:        false,
		id:             id,
	}
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	w.running = true
	defer func() { w.running = false }()
	for {
		select {
		case <-w.workerQuitChan:
			return
		case <-ctx.Done():
			return
		default:
		}
		work := w.workQueue.Pop(ctx)
		if work == nil {
			// TODO: log that we got nil and are shutting down
			log.Println("Got nil work, shutting down worker", w.id)
			return
		}
		result, err := w.executor.Do(ctx, work)
		w.resultHandler.Handle(ctx, result, err)
	}
}

func (w *Worker) Stop() {
	close(w.workerQuitChan)
}
