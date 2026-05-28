package queue

import (
	"sync"
	"sync/atomic"
	"time"
)

type Queue[T any] struct {
	head atomic.Pointer[Node[T]] // first node (dummy)
	tail atomic.Pointer[Node[T]] // last node

	nodeCount int          // max nodes pre-allocated in pool
	nodePool  sync.Pool    // pool for reusing nodes

	hazard  *hazardTable[T]    // hazard pointer table for safe reclamation
	reclaim ReclamationStack[T] // stack of retired nodes pending cleanup

	reclaimInterval time.Duration // how often cleanup runs
}

// New creates a lock-free queue with a pre-filled node pool
// and starts a background goroutine for periodic cleanup.
func New[T any]() *Queue[T] {
	const (
		defaultMaxNodes = 10000
		defaultInterval = 5 * time.Second
	)
	q := &Queue[T]{
		nodeCount:       defaultMaxNodes,
		hazard:          NewHazarTable[T](),
		reclaim:         ReclamationStack[T]{},
		reclaimInterval: defaultInterval,
		nodePool: sync.Pool{
			New: func() any { return &Node[T]{} }, // Grabs one. If pool were empty, it would call New.
		},
	}
	// Initialize all nodes and add them to pool.
	for range q.nodeCount {
		q.nodePool.Put(&Node[T]{}) // Manually creates nodes and stores them
	}
	// Setup head and tail with dummy node.
	dummyNode := q.nodePool.Get().(*Node[T])
	q.head.Store(dummyNode)
	q.tail.Store(dummyNode)

	go q.reclaimRoutine() // Background goroutine to clean up reclaimed nodes

	return q
}

// getNode from pool.
func (q *Queue[T]) getNode(value T) *Node[T] {
	node := q.nodePool.Get().(*Node[T])
	node.Val = value
	node.Next.Store(nil)
	return node
}

func (q *Queue[T]) Enqueue(value T) {
	// Protect tail with a hazard pointer, then try to link new node at its next.
	// If another thread already linked something, help advance tail forward.
	node := q.getNode(value)
	_ = node
	hp, hpIdx := q.hazard.Acquire()
	defer q.hazard.Release(hpIdx)

	for {
		tailPtr := q.tail.Load() //get current tail
		hp.ptr.Store(tailPtr)    //protect it immediately

		//Revalidate — tail may have changed before we protected it
		if tailPtr != q.tail.Load() {
			continue
		}
		nextPtr := tailPtr.Next.Load()

		if tailPtr != q.tail.Load() {
			continue
		}
		if nextPtr == nil {
			//as tail is last node link the node
			if tailPtr.Next.CompareAndSwap(nil, node) {
				q.tail.CompareAndSwap(tailPtr, node)
				return
			}
		} else {
			q.tail.CompareAndSwap(tailPtr, nextPtr)
		}

	}
}

func (q *Queue[T]) Dequeue() (T, bool) {
	// Protect head and its next with two hazard pointers.
	// If head == tail and next is nil → queue empty.
	// If head == tail but next exists → help advance tail.
	// Otherwise CAS head forward and defer the old dummy for reclamation.
	hp1, hpIdx1 := q.hazard.Acquire()
	hp2, hpIdx2 := q.hazard.Acquire()
	defer func() {
		q.hazard.Release(hpIdx1)
		q.hazard.Release(hpIdx2)
	}()

	for {
		headPtr := q.head.Load() // get current head
		hp1.ptr.Store(headPtr)   //protect it

		// Revalidate
		if headPtr != q.head.Load() {
			continue
		}

		tailPtr := q.tail.Load()
		nextPtr := headPtr.Next.Load()
		hp2.ptr.Store(nextPtr)

		if headPtr != q.head.Load() {
			continue
		}

		if headPtr == tailPtr {
			if nextPtr == nil {
				return *new(T), false
			}
			q.tail.CompareAndSwap(tailPtr, nextPtr)
		} else {
			if q.head.CompareAndSwap(headPtr, nextPtr) {
				value := nextPtr.Val
				q.deferReclamation(headPtr)
				return value, true
			}
		}
	}
}

func (q *Queue[T]) deferReclamation(node *Node[T]) { q.reclaim.Push(node) } // push retired node to reclaim stack

// reclaimRoutine runs cleanup periodically to free retired nodes.
func (q *Queue[T]) reclaimRoutine() {
	ticker := time.NewTicker(q.reclaimInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		q.cleanup()
	}
}

// cleanup pops all retired nodes from the reclaim stack.
// First pass: returns safe nodes to the pool, keeps hazardous ones aside.
// Second pass: pushes hazardous nodes back for the next cleanup cycle.
func (q *Queue[T]) cleanup() {
	ReclaimList := ReclamationStack[T]{}

	for {
		node := q.reclaim.Pop()
		if node == nil {
			break
		}

		if !q.IsNodeHazardous(node) {
			q.returnNode(node)
		} else {
			ReclaimList.Push(node)
		}
	}

	// Push back nodes we couldn't reclaim.
	for {
		node := ReclaimList.Pop()
		if node == nil {
			break
		}
		q.reclaim.Push(node)
	}
}

func (q *Queue[T]) IsNodeHazardous(node *Node[T]) bool {
	for i := range q.hazard.pointers {
		if q.hazard.pointers[i].ptr.Load() == node {
			return true
		}
	}
	return false
}

func (q *Queue[T]) returnNode(node *Node[T]) {
	node.Next.Store(nil)
	q.nodePool.Put(node)
}
