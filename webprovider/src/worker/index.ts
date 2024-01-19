import { wrap, type Remote } from "comlink";
export * from "comlink";

/** The message expected by the `readinessListener`. */
export const Ready = { ready: true };

/** Listen for a readiness message `{ ready: true }` from the Worker and call the `callback` once. */
// Comlink adds EventListeners "per request" and ignores any that don't match an
// expected UUID. So you can just listen for your own custom messages as well.
// https://github.com/GoogleChromeLabs/comlink/blob/dffe9050f63b1b39f30213adeb1dd4b9ed7d2594/src/comlink.ts#L603
export function readinessListener(worker: Worker, callback: () => void) {
  worker.addEventListener("message", function ready(event: MessageEvent<typeof Ready>) {
    if (!!event.data && event.data.ready === true) {
      worker.removeEventListener("message", ready);
      callback();
    };
  });
};

/** Return an instantiated object of a Worker, which exposes a Comlink-wrapped class.
 * #### Example:
 * ```
 * let simple = await construct(new Worker(new URL("@/worker/simple", import.meta.url)),
 *    SimpleWorker, "The Name");
 * console.log(await simple.name);
 * ```
 * 
 * The prototype includes some generic type magic to take the actual constructor parameters
 * that the exposed class expects and avoids a double-await like `await (await comlink<T>(...))(...args);`.
*/
export async function construct<T extends { new(...args: any[]): InstanceType<T> }>(
  worker: Worker,
  constructor: T,
  ...args: ConstructorParameters<typeof constructor>
): Promise<Remote<InstanceType<T>>> {
  let remoteClass = await comlink<typeof constructor>(worker);
  return await new remoteClass(...args) as any;
};

/** Wrap an existing Worker with a Comlink proxy `Remote<T>` and attach an optional
 * readiness-callback to signal when it's ready to receive requests.
 * * https://vitejs.dev/guide/features.html#web-workers
 * * https://github.com/GoogleChromeLabs/comlink
 **/
export function comlinkSync<T>(worker: Worker, ready?: () => void): Remote<T> {

  // optionally attach the callback using readiness listener
  if (ready != undefined) readinessListener(worker, ready);

  // wrap worker with comlink
  return wrap<T>(worker);

};

/** Wrap an existing Worker with a Comlink proxy `Remote<T>` asynchronously. The
 * returned Promise resolves only once the Remote is ready to receive messages.
 * * https://vitejs.dev/guide/features.html#web-workers
 * * https://github.com/GoogleChromeLabs/comlink
 * 
 * Note that this might soon be part of the Comlink library itself
 * ([comlink #635](https://github.com/GoogleChromeLabs/comlink/issues/635#issuecomment-1590972739)).
 * Until then, this is a thin wrapper.
 **/
export async function comlink<T>(worker: Worker): Promise<Remote<T>> {

  // create a promise to wait for the readiness message
  await new Promise<void>(resolve => readinessListener(worker, resolve));

  // now we're ready to wrap the worker with comlink
  return wrap<T>(worker);

};