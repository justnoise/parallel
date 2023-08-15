# parallel

A small library with some nice abstractions for creating fan-out parallel work queues.

To use this, implement the following interfaces:
* Producer: push work items onto a WorkQueue
* Executor: execute the work item
* ResultHandler: aggregate results and errors

### Other helpful things:
* WorkQueue is a minimal interface that can support various queue implementations. A channel based work queue is implemented but other implementations are possible.
* Stop execution outside the librrary by passing in a cancellable context in `ParallelRunner.Run()`.

## Example

```go

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

type SquareExecutor struct {
}

func (e *SquareExecutor) Do(ctx context.Context, ifaceVal interface{}) (interface{}, error) {
	val := ifaceVal.(int)
	square := val * val
	return square, nil
}

func main() {
	numWorkers := 4
	producer := &IntProducer{}
	executor := &SquareExecutor{}
	resultHandler := &IntResultSummer{}
	workQueue := NewChanWorkQueue(5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := NewParallelRunner(numWorkers, producer, executor, resultHandler, workQueue)
	err := runner.Run(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(resultHandler.sum)
}
```
