package queue

import "sync/atomic"

// ReclamationStack stores retired nodes that have been logically removed
// from the queue but may still be referenced by other threads via hazard pointers.
// Instead of freeing them immediately we push them here.
// A cleanup routine will free nodes that no hazard pointer targets.

type ReclamationStack[T any] struct {
	head atomic.Pointer[Node[T]] // top of the stack
}

// Push a node to stack
// Uses a CAS retry loop to handle concurrent pushes safely without locks.
func (s *ReclamationStack[T]) Push(node *Node[T]) {
	for {
		oldHead := s.head.Load()
		node.Next.Store(oldHead)
		if s.head.CompareAndSwap(oldHead, node) {
			return
		}
	}
}

// Pop
func (s *ReclamationStack[T]) Pop() *Node[T] {
	for {
		oldHead := s.head.Load()

		if oldHead == nil { // stack is empty
			return nil
		}

		if s.head.CompareAndSwap(oldHead, oldHead.Next.Load()) {
			return oldHead // successfully removed, return it
		}
		// CAS fails so retry
	}
}
