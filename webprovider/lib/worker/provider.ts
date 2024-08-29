/// <reference lib="webworker" />
declare var self: DedicatedWorkerGlobalScope | SharedWorkerGlobalScope;
export {};

import { InMemoryStorage, OpfsStorage, ProviderStorage } from "@wasimoff/storage/index.ts";
import { Messenger, WebSocketTransport } from "@wasimoff/transport/index.ts";
import { WasiWorkerPool } from "./workerpool.ts";
import { create, Message } from "@bufbuild/protobuf";
import { ProviderInfoSchema } from "@wasimoff/proto/messages_pb.ts";
import { rpchandler } from "@wasimoff/worker/rpchandler.ts";
import { expose, proxy as comlinkProxy, workerReady, transfer } from "./comlink.ts";
import { WasiTaskExecution } from "./wasiworker.ts";

/**
 *     Wasimoff Provider
 * ----------------------------------
 * 
 * This is the "entrypoint" to connect to a Broker and get started serving requests.
 * Usage would depend a bit on what environment you're running in. Some variants may
 * be removed / become unsupported in the future, if they're not worth the added
 * complexity.
 * 
 * In Deno (or another terminal-based environment outside the web) you should just
 * instantiate the class directly in the main thread. The Messenger and Storage can
 * be initialized beforehand, amended with your own reconnection handlers and what
 * have you .. and then passed inside the constructor. The comlink won't be exposed
 * if the file is not running in a Worker scope.
 * 
 * In the Web, you should generally start this file in a Worker and let it handle
 * the connection and storage concerns. In that case, the Provider is controlled
 * through comlink and events (for UI updates etc.) should be streamed with an
 * AsyncIterable. To avoid overcommitting resources when opening multiple tabs
 * and, perhaps more importantly, to get the same view in all tabs, you would
 * generally prefer a SharedWorker for the Provider. **However,** in Chrome it
 * is *not* supported to spawn nested DedicatedWorkers within a SharedWorker
 * (https://issues.chromium.org/issues/40902676 and /40695450). On Android, Chrome
 * does not support SharedWorkers (https://issues.chromium.org/issues/40290702)
 * at all. Thus, the safest approach is to always spawn in a Worker and prevent
 * further Providers from starting with the Web Lock API (which is unavailable
 * on Deno, thus this isn't a completely universal approach either).
 * 
 */

export class WasimoffProvider {

  static readonly logprefix = [ `%c Wasimoff Provider `, "color: indigo; background-color: #ccc;" ];

  constructor(
    /** maximum number of workers in the pool */
    public readonly nmax = navigator.hardwareConcurrency,
    /** connection to the broker */
    public messenger?: Messenger,
    /** storage backend for modules and artifacts */
    public storage?: ProviderStorage,
  ) {
    console.info(...WasimoffProvider.logprefix, "started in", self.constructor.name)
  };

  static async init(nmax: number, url: string, dir: string) {
    const p = new WasimoffProvider(nmax);
    await p.open(dir);
    await p.connect(url);
    return p;
  };


  // --------->  worker pool

  /** Return a comlink proxy of the worker pool. */
  async poolProxy() {
    return comlinkProxy(this.pool);
  }

  // hold the wasiworkers in a pool and use a Proxy to send pool updates automatically
  public pool = new Proxy(new WasiWorkerPool(this.nmax), {
    // trap property accesses that return methods which can change the pool length
    get: (target, prop, receiver) => {
      const traps = ["spawn", "scale", "fill", "drop", "flush", "killall"];
      const method = Reflect.get(target, prop, receiver);
      // wrap the function calls with an update to the broker
      if (typeof method === "function" && traps.includes(prop as string)) {
        return async (...args: any[]) => {
          let result = await (method as any).apply(target, args) as Promise<number>;
          try { this.sendInfo(await result); } catch { };
          return result;
        };
      } else {
        // anything else is passed through
        return method;
      };
    },
  });

  async run(id: string, task: WasiTaskExecution) {
    return this.pool.run(id, task);
  };


  // --------->  file storage

  /** Return a comlink proxy of the storage. */
  public storageProxy() {
    if (!this.storage) throw "storage does not exist yet";
    return comlinkProxy(this.storage);
  };

  /** Open a storage by URL. */
  async open(directory: string) {

    // can't close a filesystem yet
    if (this.storage !== undefined)
      throw "another storage is opened already";

    // pick either in-memory map or open OPFS
    if (directory === ":memory:") this.storage = new InMemoryStorage();
    else this.storage = await OpfsStorage.open(directory);

  };


  // --------->  messenger connections

  /** Return a comlink proxy of the messenger. */
  async messengerProxy() {
    if (!this.messenger) throw "messenger does not exist yet";
    return comlinkProxy(this.messenger);
  };

  // (re)connect to a broker by url
  async connect(url: string) {

    // close previous connections
    if (this.messenger !== undefined && !this.messenger.closed.aborted) {
      this.messenger.close("reconnecting");
    };

    // only the websocket transport is implemented so far
    if (url.match(/^wss?:\/\//) === null) throw "must be a WebSocket URL";
    const wst = WebSocketTransport.connect(url);
    this.messenger = new Messenger(wst);
    await wst.ready;

    // send current concurrency
    this.sendInfo(this.pool.length);

  };

  async disconnect() {
    if (this.messenger !== undefined && !this.messenger.closed.aborted) {
      this.messenger.close("bye");
    };
  };


  // --------->  handle rpc requests on messenger

  // bind the rpchandler function into this class
  private rpchandler = rpchandler.bind(this);

  /** Start handling RPC requests from the messenger. Await this method to be
   * notified when the connection closes because that will break the loop inside. */
  async handlerequests() {

    // storage must be opened already to register rpchandler
    if (this.storage === undefined)
      throw "need to open a storage first";

    // must have an open messenger on which to receive requests
    if (this.messenger === undefined || this.messenger.closed.aborted)
      throw "need to connect to a broker first";

    // this will loop until the messenger is closed
    for await (const request of this.messenger.requests) {
      request(request => this.rpchandler(request));
    };

  };

  /** Get a ReadableStream of the Events from the messenger. */
  async getEventstream() {

    // must have an open messenger on which to receive events
    if (this.messenger === undefined || this.messenger.closed.aborted)
      throw "need to connect to a broker first";

    // create a ReadableStream from the events iterable
    const iterator = this.messenger.events[Symbol.asyncIterator]()
    const stream = new ReadableStream<Message>({
      async pull(controller) {
        let { done, value } = await iterator.next();
        if (done) return controller.close();
        if (value) controller.enqueue(value);
      },
    });

    // transfer the stream
    return transfer(stream, [ stream ]);

  };


  // --------->  shorthands to send events

  async sendInfo(pool?: number, name?: string, useragent?: string) {
    if (this.messenger === undefined) throw "not connected yet";
    return this.messenger.sendEvent(create(ProviderInfoSchema, {
      name, useragent, pool: { concurrency: pool },
    }));
  };




};


// detect if we're running in a worker and expose the comlink interface
if (self.constructor.name === "DedicatedWorkerGlobalScope" && self instanceof DedicatedWorkerGlobalScope) {

  // in a "normal" Worker
  // locks should be handled externally, before the Worker is even started
  console.log(...WasimoffProvider.logprefix, "new dedicated Worker started");
  expose(WasimoffProvider, self);
  self.postMessage(workerReady);

} else if (self.constructor.name === "SharedWorkerGlobalScope" && self instanceof SharedWorkerGlobalScope) {

  // in a SharedWorker, listen for connections 
  console.log(...WasimoffProvider.logprefix, "new SharedWorker started");
  self.addEventListener("connect", ({ ports }) => {
    console.log(...WasimoffProvider.logprefix, "new connection");
    const port = ports[0];
    expose(WasimoffProvider, port);
    port.postMessage(workerReady);
  });

  // TODO: proper connection manager?
  // search for `Tabulator` in sharedworker.ts, somewhere before 6f0cd00

}
