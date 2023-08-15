package parallel

import (
	"context"
)

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
