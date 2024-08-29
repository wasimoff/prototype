import { defineStore } from "pinia";
import { ref } from "vue";

import { construct, Remote } from "@wasimoff/worker/comlink";
import { WasimoffProvider } from "@wasimoff/worker/provider";
import { WasiWorkerPool } from "@wasimoff/worker/workerpool";
import { ProviderStorage } from "@wasimoff/storage";
import { Messenger } from "@wasimoff/transport";
import { useTerminal } from "./terminal";

export const useProvider = defineStore("WasimoffProvider", () => {

  // whether we are currently connected to the broker
  const connected = ref(false);

  // current state of workers in the pool
  const workers = ref<boolean[]>([]);

  // update busy map on interval
  setInterval(async () => {
    // this interval slows my devtools inspector to a crawl but works fine when closed
    if ($pool.value) workers.value = await $pool.value.busy
  }, 50);

  // keep various proxies in refs ($ = Remote)
  const worker = ref<Worker>();
  const $provider = ref<Remote<WasimoffProvider>>();
  const $pool = ref<Remote<WasiWorkerPool>>();
  const $messenger = ref<Remote<Messenger>>();
  const $storage = ref<Remote<ProviderStorage>>();

  // have a terminal for logging
  const terminal = useTerminal();

  // start the worker immediately after instantiation
  navigator.locks.request("wasimoff", { ifAvailable: true }, async (lock) => {

    // fail if lock was already held in another tab
    if (lock === null) {
      const err = "another WasimoffProvider Worker is already running in another tab";
      terminal.error(`ERROR: failed to start Provider, ${err}`);
      throw err;
    }

    // start a worker and connect the comlink proxy
    connected.value = false;
    worker.value = new Worker(new URL("@wasimoff/worker/provider.ts", import.meta.url), { type: "module" });
    $provider.value = await construct<typeof WasimoffProvider>(worker.value);

    // wrap the pool proxy in another proxy to keep worker count updated
    $pool.value = new Proxy(await $provider.value.poolProxy(), {
      // trap property accesses that return methods which can change the pool length
      get: (target, prop, receiver) => {
        const traps = ["spawn", "scale", "fill", "drop", "flush", "killall"];
        const method = Reflect.get(target, prop, receiver);
        // wrap the function calls with an update to the broker
        if (typeof method === "function" && traps.includes(prop as string)) {
          return async (...args: any[]) => {
            let result = await (method as any).apply(target, args) as Promise<number>;
            try { workers.value = await target.busy; } catch { };
            return result;
          };
        } else {
          // anything else is passed through
          return method;
        };
      },
    });

    // return an infinite Promise; lock is only released when tab is closed
    return new Promise(() => { /* forever */ });
  });

  async function open(...args: Parameters<WasimoffProvider["open"]>) {
    if (!$provider.value) throw "no provider connected yet";
    // open the filesystem, get a proxy
    await $provider.value.open(...args);
    $storage.value = await $provider.value.storageProxy();
  };

  async function connect(...args: Parameters<WasimoffProvider["connect"]>) {
    if (!$provider.value) throw "no provider connected yet";
    // connect the transport (waits for readiness), get a proxy
    await $provider.value.connect(...args);
    $messenger.value = await $provider.value.messengerProxy();
    connected.value = true;
    // fill the pool it it's empty
    if (workers.value.length === 0 && $pool.value) {
      // doing it manually here is more responsive, because
      // each spawn updates the workers ref
      let capacity = await $pool.value.capacity;
      while (await $pool.value.spawn() < capacity);
    };
  };

  async function disconnect() {
    if (!$provider.value) throw "no provider connected yet";
    await $provider.value.disconnect();
    connected.value = false;
  };

  async function handlerequests() {
    if (!$provider.value) throw "no provider connected yet";
    await $provider.value.handlerequests();
    // the above promise only returns when the loop dies
    connected.value = false;
  };

  // exported as store
  return {
    // plain refs
    connected, workers,
    // comlink proxies
    $provider, $pool, $messenger, $storage,
    // special-cased methods
    open, connect, disconnect, handlerequests,
  };

});