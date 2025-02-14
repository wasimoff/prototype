import { construct, releaseProxy, type WrappedWorker } from "./comlink.ts";
import { type WasiWorker, type WasiTaskExecution, WasiTaskResult } from "./wasiworker.ts";
import { Queue } from "@wasimoff/func/queue.ts";

// colorful console logging prefix
const logprefix = [ "%c[WasiWorkerPool]", "color: purple;" ] as const;

/** Worker threadpool, which dispatches tasks to WasmWorkers. */
export class WasiWorkerPool {

  constructor(
    /** The absolute maximum number of workers in this pool. */
    public readonly capacity: number = navigator.hardwareConcurrency,
  ) {

    // test the pyodide worker
    setTimeout(async () => {
      while (true) {

        // pause as long as pool is empty
        if (this.length === 0) {
          await new Promise(r => setTimeout(r, 1000));
          continue;
        };

        // queue work on a worker but continue this loop as soon as a worker is popped
        await new Promise<void>(next => {
          this.do(async worker => {
            await worker.link.runpy(
              "testing",
              "import numpy as np; print('random mean:', np.random.rand(5,5).mean())",
              [ "numpy" ]
            );
          }, next)
        });
      }
    }, 9999999999999); // basically, never, but Infinity doesn't work

  };

  // hold the Workers in an array
  private pool: WrappedWorker<WasiWorker, {
    index: number,
    busy: boolean,
    taskid?: string,
    cancelled?: boolean,
    reject?: () => void,
  }>[] = [];
  private nextindex = 0;

  /** Get the number of Workers currently in the pool. */
  get length() { return this.pool.length; };

  /** Get a "bitmap" of busy workers. */
  get busy() { return this.pool.map(w => w.busy); };

  // an asynchronous queue to fetch an available worker
  private queue = new Queue<typeof this.pool[0]>;


  // --------->  spawn new workers

  /** Add a new Worker to the pool. */
  async spawn() {
    // TODO: serialization for multiple async calls, e.g. call spawn twice with len=cap-1

    // check for maximum size
    if (this.length >= this.capacity)
      throw "Maximum pool capacity reached!";

    // construct a new worker with comlink
    let index = this.nextindex++;
    console.info(...logprefix, "spawn Worker", index);
    const worker = new Worker(new URL("./wasiworker.ts", import.meta.url), { type: "module" });
    const link = await construct<typeof WasiWorker>(worker, index);

    // append to pool and enqueue available for work
    const wrapped = { index, worker, link, busy: false };
    this.pool.push(wrapped);
    this.queue.put(wrapped);
    return this.length;

  };

  /** Scale to a certain number of Workers is in the pool, clamped by `nmax`. */
  async scale(n: number = this.capacity) {
    n = this.clamped(n);
    if (this.length < n)
      while (this.length < n) await this.spawn();
    else
      while (this.length > n) await this.drop();
    return this.length;
  };

  /** Add Workers to maximum capacity. */
  async fill() {
    return this.scale();
  };

  // clamp a desired value to maximum number of workers
  private clamped(n?: number): number {
    if (n === undefined || n > this.capacity) return this.capacity;
    if (n <= 0) return 0;
    return n;
  };


  // --------->  terminate workers

  /** Stop a Worker gracefully and remove it from the pool. */
  async drop() {

    // exit early if pool is already empty
    if (this.length === 0) return this.length;

    // take an idle worker from the queue
    const worker = await this.queue.get();

    // remove it from the pool and release resources
    this.pool.splice(this.pool.findIndex(el => el === worker), 1);
    console.info(...logprefix, "shutdown worker", worker.index);
    worker.link[releaseProxy]();
    worker.worker.terminate();
    return this.length;

  };

  /** Terminate all Workers in the pool gracefully. */
  async flush() {
    while (await this.drop() !== 0);
    return this.length;
  };

  /** Forcefully terminate all Workers and reset the queue. */
  async killall() {
    if (this.length === 0) return;
    console.warn(...logprefix, `killing all ${this.length} workers`);
    this.pool.forEach(w => {
      w.link[releaseProxy]();
      w.worker.terminate();
    });
    this.pool = [];
    this.queue = new Queue();
    return this.length;
  };

  /** Cancel a running task. There's not really any good way of stopping an
   * execution once the WebAssembly module is started, so just terminate and
   * respawn the worker. */
  async cancel(taskid: string) {
    // find a worker executing this task id
    let w = this.pool.find(w => w.taskid === taskid);
    if (w !== undefined) {
      w.cancelled = true;
      console.warn(...logprefix, `cancel and respawn worker ${w.index}`);
      // terminate and remove from pool
      this.pool.splice(this.pool.findIndex(el => el === w), 1);
      w.link[releaseProxy]();
      w.worker.terminate();
      w.reject?.();
      // and respawn
      await this.spawn();
    };
  }


  // --------->  send tasks to workers

  /** The `run` method tries to get a free (~ non computing) worker from
   * the pool and executes a Wasi task on it. The `next` function is called
   * when a worker has been taken from the queue and before execution begins.
   * Afterwards, the method makes sure to put the worker back into the queue,
   * so *don't* keep any references to it around! The result of the computation
   * is finally returned to the caller in a Promise. */
  async run(taskid: string, task: WasiTaskExecution, next?: () => void) {

    // exit early if pool is empty
    if (this.length === 0) throw new Error("no workers in pool");

    // take an idle worker from the queue
    const worker = await this.queue.get(); next?.();
    worker.busy = true;
    worker.taskid = taskid;

    // try to execute the task and put worker back into queue
    try {
      // promise can be rejected if the task is cancelled
      return await new Promise<WasiTaskResult>((resolve, reject) => {
        worker.reject = reject;
        worker.link.run(taskid, task).then(resolve);
      });
    } finally {
      // don't requeue if it's terminated
      if (worker.cancelled !== true) {
        worker.busy = false;
        worker.taskid = undefined;
        await this.queue.put(worker);
      };
    };

  };

  async do(work: (worker: typeof this.pool[0]) => Promise<void>, next?: () => void) {

    // exit early if pool is empty
    if (this.length === 0) throw new Error("no workers in pool");

    // take an idle worker from the queue
    const worker = await this.queue.get(); next?.();
    worker.busy = true;

    try {
      return await new Promise((resolve, reject) => {
        worker.reject = reject;
        work(worker).then(resolve);
      });
    } finally {
      if (worker.cancelled !== true) {
        worker.busy = false;
        worker.taskid = undefined;
        await this.queue.put(worker);
      };
    };
  }

  // TODO: this was used to benchmark main thread vs workers
  // async race(n: number, task: WasiTaskExecution) {
  //   if (!(task.wasm instanceof WebAssembly.Module))
  //     task.wasm = await WebAssembly.compile(task.wasm);
  //   let t0 = performance.now();
  //   let promises = Array(n).fill(null).map((_, i) => this.run(`${i}`, task));
  //   await Promise.all(promises);
  //   return performance.now() - t0;
  // };

}
