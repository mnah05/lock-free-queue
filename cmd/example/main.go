package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	q "github.com/mnah05/lock-free-queue"
)

func main() {
	queue := q.New[int]()
	const numProducers = 4
	const numConsumers = 4
	const itemsPerProducer = 250

	var produced atomic.Int64
	var consumed atomic.Int64
	var wg sync.WaitGroup

	// Producers
	for i := range numProducers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			base := id * itemsPerProducer
			for j := range itemsPerProducer {
				queue.Enqueue(base + j)
				produced.Add(1)
			}
		}(i)
	}

	// Consumers — stop when we've dequeued everything
	done := make(chan struct{})
	for range numConsumers {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_, ok := queue.Dequeue()
					if ok {
						consumed.Add(1)
					}
				}
			}
		}()
	}

	wg.Wait()

	for produced.Load() != consumed.Load() {
		// spin until all items are consumed
	}

	close(done)

	fmt.Printf("produced: %d, consumed: %d\n", produced.Load(), consumed.Load())
	if produced.Load() == consumed.Load() {
		fmt.Println("OK — all items dequeued successfully")
	} else {
		fmt.Println("MISMATCH — some items lost")
	}
}
