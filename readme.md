# WHY

With this we aim to implement Michael and Scott's non-blocking concurrent queue algorithm

- paper: [here](http://dl.acm.org/doi/epdf/10.1145/248052.248106)

## Credits

Huge thanks to [**Ahrav**](https://github.com/ahrav) for their outstanding reference implementation [go-lockfree-queue](https://github.com/ahrav/go-lockfree-queue/). This project was built by studying and learning from their excellent work — all credit for the core implementation approach goes to them.

## Personal notes

- A dummy node sits at head so tail never lags behind head on an empty queue.
- Enqueue: protect tail with a hazard pointer, CAS the new node onto `tail.Next`, then advance tail.
- Dequeue: protect head and its next with two hazard pointers. If head == tail and next is nil → empty. Otherwise CAS head forward to skip the old dummy and defer it for reclamation.
- Enqueue and dequeue touch different ends so they never block each other — only producer vs producer or consumer vs consumer contend.
- Hazard pointers: a fixed-size table of slots. `Acquire` grabs a free slot, `Release` clears it. Dequeue takes two slots to protect both head and its next during the CAS.
- Reclamation stack: a Treiber stack that stores retired (logically removed) nodes.
- Cleanup runs periodically as a two-pass process — first pass separates safe nodes (return to pool) from hazardous ones (still referenced); second pass pushes hazardous ones back to the reclaim stack.
- Nodes are pre-allocated into a `sync.Pool`, retrieved by `getNode`, and returned to the pool after cleanup confirms they're safe.
