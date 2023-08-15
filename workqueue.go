package parallel

import (
	"context"
)

// WorkQueue is a minimal interface that could support a variety of implementations including
// a simple channel based queue
// A priority queue
// A rate limiting queue useful for load testing
// etc.
type WorkQueue interface {
	Push(context.Context, interface{}) error
	Pop(context.Context) interface{}
	Empty() bool
}

type ChanWorkQueue struct {
	workChan chan interface{}
	size     int
}

func NewChanWorkQueue(size int) *ChanWorkQueue {
	return &ChanWorkQueue{
		workChan: make(chan interface{}, size),
		size:     size,
	}
}

func (q *ChanWorkQueue) Push(ctx context.Context, work interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.workChan <- work:
		return nil
	}
}

func (q *ChanWorkQueue) Pop(ctx context.Context) interface{} {
	select {
	case <-ctx.Done():
		return nil
	case work := <-q.workChan:
		return work
	}
}

func (q *ChanWorkQueue) Empty() bool {
	return len(q.workChan) == 0
}
