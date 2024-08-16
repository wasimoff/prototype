/** PushableAsyncIterable is an AsyncIterable that can be "fed" from outside.
 * Multiple iterators share a buffer, so you can build n:m networks or just
 * convert an EventEmitter into an AsyncIterable. No backpressure mechanisms! */
export class PushableAsyncIterable<T> implements AsyncIterable<T> {

  private buffer: (T | Promise<T>)[] = []; // pushed items
  private waiting: ((v: void) => void)[] = []; // waiting iterators

  private closed = false; // close was called
  static ErrClosed = new Error("pushable is closed");

  /** Push an item to the buffer and wake an iterator, if one is waiting. */
  push(item: T | Promise<T>) {
    if (this.closed) throw PushableAsyncIterable.ErrClosed;
    this.buffer.push(item); // push item in the buffer
    this.waiting.shift()?.(); // maybe wake up an iterator
  };

  /** Close the Pushable, exit all iterators and prevent further insertions. */
  close() {
    this.closed = true; // signal for iterators
    this.waiting.forEach(wake => wake()); // wake all
    this.waiting = [];
  };

  /** The number of active iterators listening. */
  get listeners() { return this._listeners; }
  private _listeners = 0;
  
  // implement the protocol needed for 'for await (let el of ...)'
  async *[Symbol.asyncIterator]() { try {
    this._listeners++;
    for/*ever*/ (;;) {
      while (!this.buffer.length) { // while empty
        if (this.closed) return; // orderly exit on close
        await new Promise(wake => this.waiting.push(wake)); // sleep
      };
      yield this.buffer.shift()!; // yield first item
    };
  } finally {
    this._listeners--;
  }};

};

// TODO: remove, used for debugging
(globalThis as any).PushableAsyncIterable = PushableAsyncIterable;