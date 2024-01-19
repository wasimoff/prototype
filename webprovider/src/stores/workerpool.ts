import { computed, shallowRef, watch } from "vue";
import { defineStore } from "pinia";
import { type Remote, construct, proxy } from "@/worker";
import { WASMRunner } from "@/worker/wasmrunner";
import { useTerminal } from "./terminal";
import { useFilesystem } from "./filesystem";
import { Queue, serialize } from "@/fn/utilities";

/** Use a store as a "thread pool" of `WASMRunner` Workers to run computation requests on. */
export const useWorkerPool = defineStore("WorkerPool", () => {
  // colorful console logging prefix
  const prefix = [ "%c WorkerPool ", "background: violet; color: black;" ];

  // use other stores for terminal output and filesystem access
  const terminal = useTerminal();
  const filesystem = useFilesystem();

  // how many threads we can sensibly have at most
  const nmax = Math.max(2, window.navigator.hardwareConcurrency);

  // clamp a desired target count below the `nmax` value
  function limitedCount(n: number | "max"): number {
    if (n === "max" || n > nmax) return nmax;
    if (n <= 0) return 0;
    return n;
  };

  // keep workers and their comlinks in an array
  type Runner = { worker: Worker, link: Remote<WASMRunner> };
  const pool = shallowRef<Runner[]>([]);
  const count = computed(() => pool.value.length);
  let next = 0; // running index for worker naming

  // queue of workers awaiting requests
  let workerqueue = new Queue<Runner>();

  /** Add a new `WASMRunner` to the pool. */
  async function add() {

    // check for maximum size
    if (count.value >= nmax) { throw "Maximum pool capacity reached!"; }

    // construct a new worker with comlink
    console.debug(...prefix, "add worker", next, "to the pool");
    const worker = new Worker(new URL("@/worker/wasmrunner", import.meta.url), { type: "module" });
    const link = await construct(worker, WASMRunner, String(next++).padStart(2, "0"), proxy(terminal), proxy(filesystem), true);
    terminal.success(`New Worker: ${await link.name}!`);

    // append to pool and enqueue worker
    const wrapped = { worker, link };
    pool.value = [ ...pool.value, wrapped ];
    await workerqueue.put(wrapped);

  };

  /** Fill the pool with runners up to `nmax`. */
  async function fill() { while (count.value < nmax) await add(); }

  /** Ensure that a certain number of runners is in the pool. */
  async function ensure(n: number | "max") {
    n = limitedCount(n);
    if (count.value < n) while (count.value < n) await add();
    else while (count.value > n) await terminate();
  }

  /** Terminate a single Worker from the pool (oldest first). */
  async function terminate() {
    if (count.value > 0) {
      let wrapped = await workerqueue.get();
      let name = await wrapped.link.name;
      let index = pool.value.indexOf(wrapped);
      console.debug(...prefix, "terminate worker", `(${index}, ${name})`);
      terminal.error(`Terminating Worker (${index}, ${name}).`);
      
      pool.value.splice(index, 1);
      pool.value = [ ...pool.value ];
      wrapped.worker.terminate();
    };
  }

  /** Terminate and remove all Workers from the pool. */
  async function killall() {
    while (count.value > 0) {
      let w = pool.value[0];
      w.worker.terminate();
      pool.value = pool.value.slice(1);
    };
    terminal.error("Killed all workers!")
    workerqueue = new Queue<Runner>();
  }


  /** The `exec` method tries to get a free (~ non computing) worker from
   * the pool and executes a `task` on it. The `next` function is called
   * when a worker has been taken from the queue and before execution begins.
   * Afterwards, the method makes sure to put the worker back into the queue,
   * so *don't* keep any references to it around! The result of the computation
   * is finally returned to the caller in a Promise. */
  async function exec <Result> (task: (worker: Remote<WASMRunner>) => Promise<Result>, next?: () => void) {
    if (count.value === 0) throw new Error("no workers in pool");
    const wrapped = await workerqueue.get(); next?.();
    try {
      return await task(wrapped.link);
    } finally {
      await workerqueue.put(wrapped);
    }
  };

  // return methods for consumers
  return {
    nmax, count,
    add: serialize(add),
    terminate: serialize(terminate),
    fill: serialize(fill),
    exec,
    killall,
    ensure: serialize(ensure),
  };

});