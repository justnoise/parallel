package parallel

import (
	"context"
	"time"
)

// WorkQueue is a minimal interface that could support a variety of implementations including
// a simple channel based queue
// A rate limiting queue useful for load testing
// A priority queue
type WorkQueue interface {
	Push(context.Context, interface{}) error
	Pop(context.Context) interface{}
	Len() int
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

func (q *ChanWorkQueue) Len() int {
	return len(q.workChan)
}

func (q *ChanWorkQueue) Stop() {
	close(q.quit)
}

type RateLimitingWorkQueue struct {
	producerChan chan interface{}
	workerChan   chan interface{}
	size         int
	quit         chan bool
}

func NewRateLimitingWorkQueue(ctx context.Context, size int, interval time.Duration) *RateLimitingWorkQueue {
	wq := &RateLimitingWorkQueue{
		producerChan: make(chan interface{}, size),
		workerChan:   make(chan interface{}),
		size:         size,
		quit:         make(chan bool),
	}
	go wq.run(ctx, interval)
	return wq
}

// Tickers don't produce a perfect rate, especially at high frequency and under
// load. This function attempts to calculate the number of items we should have
// sent sent the last tick and accumulate a debt of items sent. It's not perfect
// but it gets us close to sending the right number of items in a given
// interval.
func (q *RateLimitingWorkQueue) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	debt := time.Duration(0)
	for {
		lastSend := time.Now()
		select {
		case <-ctx.Done():
			return
		case <-q.quit:
			return
		case <-ticker.C:
			start := time.Now()
			// Don't accumulate debt if there's no work to do. Producer chan is
			// usually empty at the start of a test.
			if len(q.producerChan) == 0 {
				continue
			}
			debt += time.Since(lastSend)
			neededSends := int(debt / interval)
			remainder := debt % interval
			possibleSends := len(q.producerChan)
			// Put a limit on number of items we can send to workers at once
			freeBuffer := q.size - len(q.workerChan)
			if possibleSends > freeBuffer {
				possibleSends = freeBuffer
			}
			sends := neededSends
			debt = remainder
			if neededSends > possibleSends {
				sends = possibleSends
				debt += time.Duration(neededSends-possibleSends) * interval
			}
			lastSend = start
			for i := 0; i < sends; i++ {
				val := <-q.producerChan
				select {
				case <-ctx.Done():
					return
				case <-q.quit:
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

func (q *RateLimitingWorkQueue) Len() int {
	return len(q.producerChan)
}

func (q *RateLimitingWorkQueue) Stop() {
	close(q.quit)
}
