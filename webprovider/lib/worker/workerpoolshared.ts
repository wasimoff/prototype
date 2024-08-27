import { comlink, type WrappedSharedWorker } from "./index.ts";
import { type SharedWasimoff } from "./sharedworker.ts";

/** A very thin wrapper to create a new Comlink connection to a SharedWorker holding a WasiWorkerPool inside. */
export async function SharedWasiWorkerPool(): Promise<WrappedSharedWorker<typeof SharedWasimoff, {}>> {

  const worker = new SharedWorker(new URL("./sharedworker.ts", import.meta.url), { type: "module" });
  const link = await comlink<typeof SharedWasimoff>(worker.port);

  return { worker, link };

};