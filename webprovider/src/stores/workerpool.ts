import { ref } from "vue";
import { defineStore } from "pinia";
import { useTerminal } from "./terminal";
import { serialize } from "../../lib/func/promises";
import { type WasiTaskExecution } from "../../lib/worker/wasiworker.ts";
import { SharedWasiWorkerPool } from "../../lib/worker/workerpoolshared.ts";

/** Use a store as a "thread pool" of `WASMRunner` Workers to run computation requests on. */
export const useWorkerPool = defineStore("WorkerPool", () => {
  // colorful console logging prefix
  const prefix = [ "%c WorkerPool ", "background: violet; color: black;" ];

  // use other stores for terminal output and filesystem access
  const terminal = useTerminal();

  // get a connection to the SharedWorker
  let pool: Awaited<ReturnType<typeof SharedWasiWorkerPool>> | undefined;
  (async () => {
    pool = await SharedWasiWorkerPool();
    let spawn = pool.link.__spawned; let n = fill();
    terminal.success(`SharedWasiWorkerPool connected! { spawned: ${await spawn}, workers: ${await n} }`);
  })();

  // how many threads we can sensibly have at most
  const nmax = Math.max(2, window.navigator.hardwareConcurrency);

  // keep workers and their comlinks in an array
  const count = ref(0);

  async function add() {
    if (pool === undefined) throw "pool not connected yet";
    try {
      let name = await pool.link.spawn();
      console.debug(...prefix, "added worker", name, "to the pool");
      // terminal.success(`New Worker: ${await link.name}!`);
      count.value += 1;
      return name;
    } catch(err) {
      console.error("couldn't spawn worker:", err);
      terminal.error("couldn't spawn Worker: " + String(err));
    };
  };

  async function fill() {
    if (pool === undefined) throw "pool not connected yet";
    let n = await pool?.link.fill();
    count.value = n;
    return n;
  }

  /** Ensure that a certain number of runners is in the pool. */
  async function ensure(n: number | "max") {
    if (pool === undefined) throw "pool not connected yet";
    return count.value = await pool.link.scale(n);
  }

  /** Terminate a single Worker from the pool (oldest first). */
  async function terminate() {
    if (pool === undefined) throw "pool not connected yet";
    let name = await pool.link.drop();
    if (name === undefined) return;
    console.debug(...prefix, "terminated worker", name);
    terminal.error(`Terminated Worker ${name}.`);
    count.value -= 1;
  }

  /** Terminate and remove all Workers from the pool. */
  async function killall() {
    if (pool === undefined) throw "pool not connected yet";
    await pool.link.killall();
    count.value = 0;
    terminal.error("Killed all workers!");
  };


  /** The `exec` method tries to get a free (~ non computing) worker from
   * the pool and executes a `task` on it. The `next` function is called
   * when a worker has been taken from the queue and before execution begins.
   * Afterwards, the method makes sure to put the worker back into the queue,
   * so *don't* keep any references to it around! The result of the computation
   * is finally returned to the caller in a Promise. */
  // async function exec <Result> (task: (worker: Remote<WASMRunner>) => Promise<Result>, next?: () => void) {
  //   if (count.value === 0) throw new Error("no workers in pool");
  //   const wrapped = await workerqueue.get(); next?.();
  //   try {
  //     return await task(wrapped.link);
  //   } finally {
  //     await workerqueue.put(wrapped);
  //   }
  // };

  async function run(id: string, task: WasiTaskExecution, next?: () => void) {
    if (next) next(); // TODO, without function
    if (pool === undefined) throw "pool not connected yet";
    return await pool.link.run(id, task);
  };

  // return methods for consumers
  return {
    nmax, count,
    add: serialize(add),
    terminate: serialize(terminate),
    fill: serialize(fill),
    run,
    killall,
    ensure: serialize(ensure),
  };

});