package queue

import (
	"runtime"
	"sync/atomic"
)

type Node[T any] struct {
	Val  T
	Next atomic.Pointer[Node[T]] // link to next node
}

type hazardPtr[T any] struct{ ptr atomic.Pointer[Node[T]] } //each pointer targets a node

// hazardTable manages all hazard pointers.
type hazardTable[T any] struct{ pointers []hazardPtr[T] } // a hash table to get node a space to check its availability

func NewHazarTable[T any]() *hazardTable[T] {
	maxHazardPointers := 2 * runtime.GOMAXPROCS(0) * 2 // 2 pointers per thread + buffer
	return &hazardTable[T]{pointers: make([]hazardPtr[T], maxHazardPointers)}
}

// Acquire a hazard pointer
func (ht *hazardTable[T]) Acquire() (*hazardPtr[T], int) {
	for i := range ht.pointers {
		if ht.pointers[i].ptr.Load() == nil {
			return &ht.pointers[i], i
		}
	}
	panic("no free hazard pointers")
}

// Release a hazard pointer.
func (ht *hazardTable[T]) Release(index int) {
	ht.pointers[index].ptr.Store(nil)
}
