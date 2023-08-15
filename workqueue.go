package parallel

import (
	"context"
	"time"
)

// WorkQueue is a minimal interface that could support a variety of implementations including
// a simple channel based queue
// A rate limiting queue useful for load testing
// Maybe: A priority queue
type WorkQueue interface {
	Push(context.Context, interface{}) error
	Pop(context.Context) interface{}
	Empty() bool
	// Stop is needed to ensure workers don't get stuck waiting for work
	// when there is no more work to be done
	Stop()
}

type ChanWorkQueue struct {
	workChan chan interface{}
	size     int
	quit     chan bool
}

func NewChanWorkQueue(size int) *ChanWorkQueue {
	return &ChanWorkQueue{
		workChan: make(chan interface{}, size),
		size:     size,
		quit:     make(chan bool),
	}
}

func (q *ChanWorkQueue) Push(ctx context.Context, work interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.quit:
		return nil
	case q.workChan <- work:
		return nil
	}
}

func (q *ChanWorkQueue) Pop(ctx context.Context) interface{} {
	select {
	case <-ctx.Done():
		return nil
	case <-q.quit:
		return nil
	case work := <-q.workChan:
		return work
	}
}

func (q *ChanWorkQueue) Empty() bool {
	return len(q.workChan) == 0
}

func (q *ChanWorkQueue) Stop() {
	close(q.quit)
}

// TODO: Needs untested
type RateLimitingWorkQueue struct {
	producerChan chan interface{}
	workerChan   chan interface{}
	quit         chan bool
}

func NewRateLimitingWorkQueue(ctx context.Context, size int, interval time.Duration) *RateLimitingWorkQueue {
	wq := &RateLimitingWorkQueue{
		producerChan: make(chan interface{}, size),
		workerChan:   make(chan interface{}),
		quit:         make(chan bool),
	}
	go wq.run(ctx, interval)
	return wq
}

func (q *RateLimitingWorkQueue) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case <-ctx.Done():
				return
			case val := <-q.producerChan:
				select {
				case <-ctx.Done():
					return
				case q.workerChan <- val:
				}
			}
		}
	}
}

func (q *RateLimitingWorkQueue) Push(ctx context.Context, work interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.quit:
		return nil
	case q.producerChan <- work:
		return nil
	}
}

func (q *RateLimitingWorkQueue) Pop(ctx context.Context) interface{} {
	select {
	case <-ctx.Done():
		return nil
	case <-q.quit:
		return nil
	case work := <-q.workerChan:
		return work
	}
}

func (q *RateLimitingWorkQueue) Empty() bool {
	return len(q.producerChan) == 0
}

func (q *RateLimitingWorkQueue) Stop() {
	close(q.quit)
}
