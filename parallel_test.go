package parallel

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	numWorkers int = 4
)

type IntProducer struct{}

func (p *IntProducer) Produce(ctx context.Context, workQueue WorkQueue) error {
	for i := 1; i <= 100; i++ {
		err := workQueue.Push(ctx, i)
		if err != nil {
			return err
		}
	}
	return nil
}

type IntResultSummer struct {
	sum    int
	errors int
	mu     sync.Mutex
}

func (s *IntResultSummer) Handle(ctx context.Context, result interface{}, err error) {
	s.mu.Lock()
	if err != nil {
		s.errors++
	}
	s.sum += result.(int)
	s.mu.Unlock()
}

type SquareExecutor struct{}

func (e *SquareExecutor) Do(ctx context.Context, ifaceVal interface{}) (interface{}, error) {
	val := ifaceVal.(int)
	square := val * val
	return square, nil
}

func TestParallel(t *testing.T) {
	numWorkers := 4
	producer := &IntProducer{}
	executors := make([]Executor, numWorkers)
	for i := 0; i < numWorkers; i++ {
		executors[i] = &SquareExecutor{}
	}
	resultHandler := &IntResultSummer{}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := NewParallelRunner(producer, executors, resultHandler, workQueue)
	err := runner.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 338350, resultHandler.sum)
	for _, worker := range runner.workers {
		assert.False(t, worker.running.Load())
	}
}

type ErrorProducer struct {
	maxItems int
}

func (p *ErrorProducer) Produce(ctx context.Context, workQueue WorkQueue) error {
	for i := 1; i <= 100; i++ {
		err := workQueue.Push(ctx, i)
		if err != nil {
			return err
		}
		if i > p.maxItems {
			return fmt.Errorf("error")
		}
	}
	return nil
}

func TestParallelWithProducerErrors(t *testing.T) {
	producer := &ErrorProducer{maxItems: 10}
	executors := make([]Executor, numWorkers)
	for i := 0; i < numWorkers; i++ {
		executors[i] = &SquareExecutor{}
	}
	resultHandler := &IntResultSummer{}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := NewParallelRunner(producer, executors, resultHandler, workQueue)
	err := runner.Run(ctx)
	assert.Error(t, err)
	for _, worker := range runner.workers {
		assert.False(t, worker.running.Load())
	}
}

type MaxValSummer struct {
	cancel context.CancelFunc
	maxVal int
	sum    int
	mu     sync.Mutex
}

func (s *MaxValSummer) Handle(ctx context.Context, result interface{}, err error) {
	s.mu.Lock()
	s.sum += result.(int)
	if s.sum > s.maxVal {
		s.cancel()
	}
	s.mu.Unlock()
}

// This test shows how a context passed into the result handler can be used to cancel the run.
func TestCancelRunFromResultHandler(t *testing.T) {
	numWorkers := 4
	producer := &IntProducer{}
	executors := make([]Executor, numWorkers)
	for i := 0; i < numWorkers; i++ {
		executors[i] = &SquareExecutor{}
	}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultHandler := &MaxValSummer{
		cancel: cancel,
		maxVal: 1000,
	}
	runner := NewParallelRunner(producer, executors, resultHandler, workQueue)
	err := runner.Run(ctx)
	assert.Error(t, err)
	for _, worker := range runner.workers {
		assert.False(t, worker.running.Load())
	}
}
