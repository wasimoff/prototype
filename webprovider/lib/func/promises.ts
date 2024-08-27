/** An *upside-down* Promise, which can be signalled to resolve from outside. */
export function Promised<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason: any) => void;
  let promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
};

/** A shorthand for `Promised<void>` where you are only interested in the signal itself. */
export function Signal() { return Promised<void>(); }

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
