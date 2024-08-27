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