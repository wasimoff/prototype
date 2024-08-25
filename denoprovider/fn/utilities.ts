/** An *upside-down* Promise, which can be signalled to resolve from outside. */
export function Promised<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason: any) => void;
  let promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
};

/** A shorthand for `Promised<void>` where you are only interested in the signal itself. */
export function Signal() { return Promised<void>(); }

/** Create a `ReadableStream` from an asnyc iterator. */
// https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream#convert_async_iterator_to_stream
export function toStream<T>(iterator: AsyncIterator<T, T, T>) {
  return new ReadableStream({
    async pull(controller) {
      const { done, value } = await iterator.next();
      if (done) controller.close();
      controller.enqueue(value);
    },
  });
};

/** Create an `AsyncGenerator` from a `ReadableStream`, which prevents cancelling the stream on release. */
export async function* agen<T>(stream: ReadableStream<T>): AsyncGenerator<T, void, undefined> {
  const reader = stream.getReader();
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) return;
      yield value;
    };
  } finally {
    reader.releaseLock();
  };
};

/** Get the next element from an async iterator. */
export async function next<T>(iterator: AsyncIterator<T, T, T>): Promise<T>;
/** Get the next element from a `ReadableStream`. */
export async function next<T>(stream: ReadableStream<T>): Promise<T>;

export async function next<T>(source: AsyncIterator<T, T, T> | ReadableStream<T>) {
  const EOF = () => new Error("unexpected end of iteration");
  if (source instanceof ReadableStream) {
    const reader = source.getReader();
    const { done, value } = await reader.read();
    reader.releaseLock();
    if (done) throw EOF();
    return value;
  } else {
    const { done, value } = await source.next();
    if (done) throw EOF();
    return value;
  };
};

/** Log each chunk of a stream to the console and pass it through. */
export async function* chunkLogger(stream: ReadableStream<Uint8Array>, prefix: string = "CHUNK") {
  for await (const chunk of stream) {
    console.log(`%c ${prefix} `, "background: #222; color: white;", chunk);
    yield chunk;
  };
};

/** Yield two consecutive elements from an async iterator together. */
export async function* pairs<L extends {}, R extends {}>(stream: ReadableStream<L | R> | AsyncGenerator<L | R>) {
  let tmp: L | null = null;
  for await (const obj of stream) {
    if (tmp === null) {
      tmp = obj as L;
    } else {
      yield { 0: tmp, 1: obj as R };
      tmp = null;
    };
  };
  if (tmp !== null) {
    throw new Error("unexpected end of stream with an element still in `tmp`");
  };
};


/** A simple asynchronous "FIFO" queue class. */
// Heavily inspired by github.com/NicoAdrian/async-fifo-queue, but adapted
// with generic types for Typescript and a simpler unbounded queue.
// Original Copyright (c) 2020 NicoAdrian (MIT Licensed)
export class Queue<T> {
  private prefix = (op: string) => [ `%c QUEUE ${op} `, "background: lime; color: black;" ];

  // resolve functions of waiting getters
  private getters: ((item: T) => void)[] = [];

  // the queued items
  private items: T[] = [];

  // yield an item from the queue
  async get(): Promise<T> {
    // console.debug(...this.prefix("-GET"), { items: this.items.length, waiting: this.getters.length });
    // if the queue is empty, append ourselves as a waiting promise
    if (this.items.length === 0) {
      return await new Promise<T>(r => this.getters.push(r));
    }
    // otherwise yield an element immediately
    return this.items.shift()!;
  };

  // put an item into the queue
  async put(item: T): Promise<void> {
    // console.debug(...this.prefix("+PUT"), { items: this.items.length, waiting: this.getters.length });
    // if there are getters waiting, resolve the first
    if (this.getters.length > 0) {
      return this.getters.shift()!(item);
    }
    // otherwise append to queue
    this.items.push(item);
  };

}

/** Serialize multiple asynchronous function calls. Taken from:
 * https://advancedweb.hu/how-to-serialize-calls-to-an-async-function/ */
export function serialize<T extends (...args: any[]) => ReturnType<T>>(func: T): typeof func {
 let chain: Promise<void | ReturnType<T>> = Promise.resolve();
 return ((...args: Parameters<typeof func>) => {
   const link = chain.then(() => func(...args));
   chain = link.catch(() => {});
   return link;
 }) as any; // TODO: can't quite figure out proper type inferencing here
};
