import { comlink, type WrappedSharedWorker } from "@/workerpool";
import { type SharedWasimoff } from "@/workerpool/sharedworker";

/** A very thin wrapper to create a new Comlink connection to a SharedWorker holding a WasiWorkerPool inside. */
export async function SharedWorkerPool(): Promise<WrappedSharedWorker<typeof SharedWasimoff, {}>> {

  const worker = new SharedWorker(new URL("@/workerpool/sharedworker", import.meta.url), { type: "module" });
  const link = await comlink<typeof SharedWasimoff>(worker.port);

  return { worker, link };

};