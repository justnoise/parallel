package parallel

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type IntProducer struct {
}

func (p *IntProducer) Produce(ctx context.Context, workQueue WorkQueue) error {
	for i := 1; i <= 100; i++ {
		err := workQueue.Push(ctx, i)
		if err != nil {
			return err
		}
	}
	return nil
}

type ErrorProducer struct {
}

func (p *ErrorProducer) Produce(ctx context.Context, workQueue WorkQueue) error {
	for i := 1; i <= 100; i++ {
		err := workQueue.Push(ctx, i)
		if err != nil {
			return err
		}
		if i > 10 {
			return fmt.Errorf("error")
		}
	}
	return nil
}

type IntResultSummer struct {
	sum    int
	errors int
}

func (s *IntResultSummer) Handle(ctx context.Context, result interface{}, err error) {
	if err != nil {
		s.errors++
	}
	s.sum += result.(int)
}

type IntExecutor struct {
}

func (e *IntExecutor) Do(ctx context.Context, ifaceVal interface{}) (interface{}, error) {
	val := ifaceVal.(int)
	return val, nil
}

func TestParallel(t *testing.T) {
	numWorkers := 4
	producer := &IntProducer{}
	executor := &IntExecutor{}
	resultHandler := &IntResultSummer{}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := NewParallelRunner(numWorkers, producer, executor, resultHandler, workQueue)
	err := runner.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 5050, resultHandler.sum)
}

func TestParallelWithProducerErrors(t *testing.T) {
	numWorkers := 4
	producer := &ErrorProducer{}
	executor := &IntExecutor{}
	resultHandler := &IntResultSummer{}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := NewParallelRunner(numWorkers, producer, executor, resultHandler, workQueue)
	err := runner.Run(ctx)
	assert.Error(t, err)
	for _, worker := range runner.workers {
		assert.False(t, worker.running)
	}
}
