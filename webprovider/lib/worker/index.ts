import { wrap, type Endpoint, type Remote } from "comlink";
export * from "comlink";

/** Return an instantiated object of a Worker, which exposes a Comlink-wrapped class.
 * #### Example:
 * ```
 * import { SimpleWorker } from "@/worker/simple";
 * let worker = new Worker(new URL("@/worker/simple", import.meta.url));
 * let simple = await construct(worker, SimpleWorker, "MyName");
 * console.log(await simple.name); // "MyName"
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
  // wrap the comlink proxy when it's ready
  let proxiedClass = await comlink<typeof constructor>(worker);
  // call the remote class constructor
  return await new proxiedClass(...args) as any;
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
export async function comlink<T>(endpoint: Endpoint): Promise<Remote<T>> {
  // create a promise to wait for the readiness message
  await new Promise(r => whenready(endpoint, r));
  // now we're ready to wrap the worker with comlink
  return wrap<T>(endpoint);
};

/** Listen for a readiness message `{ ready: true }` from the Worker and call the `callback` once. */
// Comlink adds EventListeners "per request" and ignores any that don't match an
// expected UUID. So you can just listen for your own custom messages as well.
// https://github.com/GoogleChromeLabs/comlink/blob/dffe9050f63b1b39f30213adeb1dd4b9ed7d2594/src/comlink.ts#L603
export function whenready(endpoint: Endpoint, callback: (u?: unknown) => void) {
  const controller = new AbortController();
  endpoint.addEventListener("message", (ev: any) => {
    if (!!ev.data && (<MessageEvent<typeof workerReady>>ev).data.ready === true) {
      controller.abort();
      callback();
    };
  }, { signal: controller.signal });
  if (endpoint.start) { endpoint.start() };
};

/** The message expected by the `readinessListener`. */
export const workerReady = { ready: true } as const;

/** A Comlink-wrapped Worker, which also still holds the bare Worker reference. */
export type WrappedWorker<Exposed, Metadata extends Object> = {
  worker: Worker,
  link: Remote<Exposed>,
} & Metadata;

/** A Comlink-wrapped SharedWorker, which also still holds the bare SharedWorker reference. */
export type WrappedSharedWorker<Exposed, Metadata extends Object> = {
  worker: SharedWorker,
  link: Remote<Exposed>,
} & Metadata;
