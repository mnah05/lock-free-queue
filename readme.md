# WHY

With this we aim to implement Michael and Scott's non-blocking concurrent queue algorithm

- paper: [here](http://dl.acm.org/doi/epdf/10.1145/248052.248106)

## Credits

Huge thanks to [**Ahrav**](https://github.com/ahrav) for their outstanding reference implementation [go-lockfree-queue](https://github.com/ahrav/go-lockfree-queue/). This project was built by studying and learning from their excellent work — all credit for the core implementation approach goes to them.

## Personal notes

- Tail only moves forward on writes (enqueue) — it always chases the latest inserted node.
- Head only moves forward on reads (dequeue) — it advances one step every time a value is consumed.

Since enqueue and dequeue touch different ends, they can now run truly in parallel. A producer and consumer never block each other — only producer vs producer, or consumer vs consumer contend.
This is simpler than the lock-free version and already a massive improvement over a single lock. But it still blocks — if two producers race, one waits. The non-blocking version eliminates even that.
