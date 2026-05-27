package queue

import "sync/atomic"

//Reclamation stack is a place to store retired nodes which are released but can
//still be read
//so instead of freeing them instantly we store in a stack
//periodically a function will run and ensure that when no thread is accessing
//a node it will get back to node pool :)

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
		//CAS failes so retry
	}
}
