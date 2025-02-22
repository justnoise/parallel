# parallel

A library with some nice abstractions for creating fan-out parallel work queues and load testers.

### Other helpful things:
* WorkQueue is a minimal interface that can support various queue implementations. A channel based work queue and rate limited queue are implemented but it could be extended to support a priority queue as well.
* Stop execution outside the librrary by passing in a cancellable context to `ParallelRunner.Run()`.

## Usage

To use this, implement the following interfaces:
* Producer: Implement `Produce` to push work items onto a WorkQueue
* Executor: Implement `Do` to execute the work item
* ResultHandler: Implement `Handle` to aggregate results and errors

Values in the pipeline are passed through as `interface{}`. Client implementations of Executor and ResultHandler must cast these to the correct types.

```
                                        ┌───────────────┐
                                        │Executor       │
                                        ├───────────────┤
                                  ┌────►│Do(interface{})├───┐
                                  │     └───────────────┘   │
┌────────────┐     ┌───────────┐  │                         │
│ Producer   │     │WorkQueue  │  │     ┌───────────────┐   │   ┌──────────────┐
├────────────┤     ├───────────┤  │     │Executor       │   │   │ResultHandler │
│ Produce()  ├────►│ Push()    │  │     ├───────────────┤   │   ├──────────────┤
└────────────┘     │ Pop()     ├──┼────►│Do(interface{})├───┼──►│Handle()      │
                   │ Len()     │  |     └───────────────┘   │   └──────────────┘
                   │ Stop()    │  |                         │
                   └───────────┘  │     ┌───────────────┐   │
                                  │     │Executor       │   │
                                  │     ├───────────────┤   │
                                  └────►│Do(interface{})├───┘
                                        └───────────────┘
```

## Example

```go

type IntProducer struct {}

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

type SquareExecutor struct {}

func (e *SquareExecutor) Do(ctx context.Context, ifaceVal interface{}) (interface{}, error) {
	val := ifaceVal.(int)
	square := val * val
	return square, nil
}

func main() {
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
	if err != nil {
		panic(err)
	}
	fmt.Println(resultHandler.sum)	// Prints 338350
}
```
