<script setup lang="ts">

import Terminal from "@app/components/Terminal.vue";
import Controls from "@app/components/Controls.vue";

const title = "wasimoff";

import { useTerminal, LogType } from "@app/stores/terminal.ts";
const terminal = useTerminal();
terminal.log(`Hello, ${title}!`, LogType.Black);



import { comlink } from "@wasimoff/worker/index.ts";
import { SharedWasimoff } from "@wasimoff/worker/sharedworker.ts";
import { WasiWorkerPool } from "@wasimoff/worker/workerpool.ts";

// simple performance timer function
async function timed<T>(fn: () => Promise<T>) {
  let t0 = performance.now();
  let result = await fn();
  return { result, duration: performance.now() - t0 };
};

let hmrController = new AbortController();
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    terminal.error("HMR! Cancel hmrController ..");
    hmrController.abort();
  });
}


// run various tests locally against different worker pools
let benchmark = async () => {

  // parameters for the bursty race
  const n = 10; const tsp = "4";
  terminal.warn(`Setup ...`);

  // listen to worker messages on broadcast channel and print to terminal?
  // if (true) {
  //   terminal.log(`Setup BroadcastChannel listener ...`);
  //   let bc = new BroadcastChannel("WasiWorkerBroadcast");
  //   bc.addEventListener("message", ({ data }: { data: SomeWasiWorkerMessage }) => {
  //     if (data.type === "cmdline") terminal.info(`WasiWorker ${data.name}: ${data.payload.id} ${data.payload.cmdline}`);
  //   }, { signal: hmrController.signal });
  // };

  // prepare task payload
  //! use ArrayBuffer since we can't seem to transfer WebAssembly.Modules to SharedWorkers
  let wasm = await (await fetch("/tsp.wasm")).arrayBuffer();
  let task = { wasm, argv: [ "tsp.wasm", "rand", tsp ], envs: [ "PROJECT=wasimoff" ] };


  // setup the SharedWorker pool
  const sharedWorker = new SharedWorker(new URL("./sharedworker.ts", import.meta.url), { type: "module" });
  const sharedLink = await comlink<typeof SharedWasimoff>(sharedWorker.port);
  const sharedFill = await timed(() => sharedLink.fill());
  
  // setup the direct pool
  const localPool = new WasiWorkerPool(Math.max(2, window.navigator.hardwareConcurrency));
  hmrController.signal.addEventListener("abort", () => localPool.killall());
  const localFill = await timed(() => localPool.fill());
  
  // remotely trigger a RACE!
  terminal.warn(`Race!`);
  let sharedRace = await timed(() => sharedLink.race(n, task));
  terminal.log(`SharedWorkerPool race done.`);
  let localRace = await timed(() => localPool.race(n, task));
  terminal.log(`LocalWorkerPool race done.`);

  terminal.success(`SharedWorkerPool filled with ${sharedFill.result} workers in ${sharedFill.duration.toFixed(1)} ms`);
  terminal.success(`LocalWorkerPool filled with ${localFill.result} workers in ${localFill.duration.toFixed(1)} ms`);
  terminal.warn(`SharedWorkerPool raced: ${sharedRace.result.toFixed(1)}/${sharedRace.duration.toFixed(1)} ms`);
  terminal.warn(`LocalWorkerPool raced: ${localRace.result.toFixed(1)}/${localRace.duration.toFixed(1)} ms`);

}; // benchmark();


</script>

<template>

  <!-- logo and title -->
  <h1 class="title">
    <img alt="WebAssembly Logo" class="logo" src="./assets/wasm.svg" />
    {{ title }}
  </h1>

  <!-- raised card for controls and terminal-->
  <div class="field box">

    <!-- worker pool and transport controls -->
    <Controls/>

    <!-- virtual console for worker output -->
    <label class="label has-text-grey-dark">Terminal <a @click="terminal.clear()" title="Clear messages">×</a></label>
    <Terminal/>

  </div>

</template>

<style scoped>

.title {
  color: rgb(75, 9, 180);
  font-weight: 500;
  font-size: 2.6rem;
  top: -10px;
}

.logo {
  position: relative;
  top: 0.7rem;
  height: 3rem;
}

</style>
