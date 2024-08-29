import { defineStore } from "pinia";
import { ref } from "vue";

import { construct, Remote } from "@wasimoff/worker/comlink";
import { WasimoffProvider } from "@wasimoff/worker/provider";
import { WasiWorkerPool } from "@wasimoff/worker/workerpool";
import { ProviderStorage } from "@wasimoff/storage";
import { Messenger } from "@wasimoff/transport";
import { useTerminal } from "./terminal";
import { useConfiguration } from "./configuration";

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

  // load configuration values
  const config = useConfiguration();

  // check if we're running exclusively (not open in another tab)
  const exclusive = new Promise<void>(resolve => {
    if ("locks" in navigator) {
      navigator.locks.request("wasimoff", { ifAvailable: true }, async (lock) => {
        if (lock === null) {
          return terminal.error("ERROR: another Provider is already running; refusing to start!");
        };
        // got the lock, continue startup
        resolve();
        // return an "infinite" Promise; lock is only released when tab is closed
        return new Promise(r => window.addEventListener("beforeunload", r));
      });
    } else {
      // can't check the lock, warn about it and continue anyway
      terminal.warn("WARNING: Web Locks API not available; can't check for exclusive Provider!");
      resolve();
    };
  });

  // start the worker when the lock has been acquired
  exclusive.then(async () => {

    // start a worker and connect the comlink proxy
    connected.value = false;
    worker.value = new Worker(new URL("@wasimoff/worker/provider.ts", import.meta.url), { type: "module" });
    $provider.value = await construct<typeof WasimoffProvider>(worker.value, config.workers);

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

    // try to grab a wakelock to keep screen on
    if ("wakeLock" in navigator) {
      try {
        const lock = await navigator.wakeLock.request("screen");
        terminal.info("Acquired a wakelock.")
        lock.addEventListener("release", () => terminal.warn("Wakelock was revoked!"));
        window.addEventListener("beforeunload", () => lock.release());
      } catch (err) {
        terminal.warn(`Could not acquire wakelock: ${err}`);
      };
    } else {
      terminal.info("Wakelock API unavailable.");
    }

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