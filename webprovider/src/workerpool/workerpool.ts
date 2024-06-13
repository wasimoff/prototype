import { type Remote, construct, releaseProxy } from "@/workerpool";
import { type SomeWasiWorkerMessage, WasiWorker, type WasiTaskExecution } from "@/workerpool/wasiworker";
import { Queue } from "@/fn/utilities";
import { Observable, Subject } from "observable-fns";

/** Worker threadpool, which dispatches tasks to WasmWorkers. */
export class WasiWorkerPool {

  constructor(
    /** The maximum number of workers in this pool. */
    private readonly nmax: number = Math.max(2, window.navigator.hardwareConcurrency),
  ) { };

  // colorful console logging prefix
  private readonly logprefix = [ "%c WasmWorkerPool ", "background: violet; color: black;" ];

  // TODO: make proper event emitter?
  private channel = new BroadcastChannel("WasiWorkerBroadcast");
  public events = new Observable<SomeWasiWorkerMessage>(subscriber => {
    this.channel.addEventListener("message", ({ data }) => subscriber.next(data as SomeWasiWorkerMessage));
  });

  // hold the Workers in an array
  private pool: WrappedWorker[] = [];
  public get workers() { return this.pool.map(p => p.name); };
  private nextindex = 0;

  // an asynchronous queue to fetch an available worker
  // TODO: rather set properties on WrappedWorker { busy: bool } atomically and use a filter?
  private queue = new Queue<WrappedWorker>;


  /** The `exec` method tries to get a free (~ non computing) worker from
   * the pool and executes a `task` on it. The `next` function is called
   * when a worker has been taken from the queue and before execution begins.
   * Afterwards, the method makes sure to put the worker back into the queue,
   * so *don't* keep any references to it around! The result of the computation
   * is finally returned to the caller in a Promise. */
  private async exec <Result> (task: (worker: Remote<WasiWorker>) => Promise<Result>, next?: () => void) {
    if (this.pool.length === 0) throw new Error("no workers in pool");
    // console.warn(...this.logprefix, "EXEC: fetch a worker");
    const worker = await this.queue.get(); next?.();
    try {
      // console.warn(...this.logprefix, "EXEC: ruuuun the task");
      let r = await task(worker.link);
      // console.warn(...this.logprefix, "EXEC: finished the task");
      return r;
    } finally {
      // console.warn(...this.logprefix, "EXEC: put worker back in queue");
      await this.queue.put(worker);
    };
  };

  /** More limited form of `exec`, which only runs `WasmWorker.run` tasks. */
  async run(id: string, task: WasiTaskExecution, next?: () => void) {
    return this.exec(w => w.run(id, task), next);
  };


  /** Add a new WasmWorker to the pool. */
  async spawn() { // TODO: re-add serialization for multiple async calls?

    // check for maximum size
    if (this.pool.length >= this.nmax) { throw "Maximum pool capacity reached!"; }

    // construct a new worker with comlink
    let name = String(this.nextindex++).padStart(3, "0");
    console.info(...this.logprefix, "add worker", name, "to the pool");
    const worker = new Worker(new URL("@/workerpool/wasiworker", import.meta.url), { type: "module" });
    const link = await construct(worker, WasiWorker, name);

    // append to pool and enqueue worker
    const wrapped = { name, worker, link };
    this.pool.push(wrapped); // TODO: make observable?
    this.queue.put(wrapped);

  };

  /** Terminate and drop a Worker from the pool. */
  async drop() {
    if (this.pool.length === 0) return;
    const w = await this.queue.get();
    this.pool.splice(this.pool.findIndex(el => el === w), 1); // TODO: make observable?
    console.info(...this.logprefix, "terminate worker", w.name, "from pool");
    w.link[releaseProxy]();
    w.worker.terminate();
  }

  /** Add Workers to maximum capacity. */
  async fill() {
    while (this.pool.length < this.nmax) await this.spawn();
    return this.pool.length;
  };

  /** Terminate all Workers in the pool gracefully. */
  async flush() { while (this.pool.length) await this.drop(); };

  /** Forcefully terminate all Workers and reset the queue. */
  killall() {
    if (this.pool.length === 0) return;
    console.info(...this.logprefix, "killing all workers:", this.workers);
    this.pool.forEach(w => {
      w.link[releaseProxy]();
      w.worker.terminate();
    });
    this.pool = [];
    this.queue = new Queue();
  };

  // // clamp a desired value to maximum number of workers
  // private clamped(n: number | "nmax"): number {
  //   if (n === "nmax" || n > this.nmax) return this.nmax;
  //   if (n <= 0) return 0;
  //   return n;
  // };





  async race(n: number, task: WasiTaskExecution) {
    if (!(task.wasm instanceof WebAssembly.Module))
      task.wasm = await WebAssembly.compile(task.wasm);
    let t0 = performance.now();
    let promises = Array(n).fill(null).map((_, i) => this.run(`${i}`, task));
    await Promise.all(promises);
    return performance.now() - t0;
  };

}

/** Reference to a WasmWorker with both the Web Worker API and Comlink proxy. */
type WrappedWorker = { name: string, worker: Worker, link: Remote<WasiWorker> };
