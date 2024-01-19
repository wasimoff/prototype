<script setup lang="ts">
import { ref, computed } from "vue";

// terminal for logging
import { useTerminal, LogType } from "@/stores/terminal";
const terminal = useTerminal();

// configuration via url fragment
import { useConfiguration } from "@/stores/configuration";
const conf = useConfiguration();

// filesystem storage
import { useFilesystem } from "@/stores/filesystem";
const opfs = useFilesystem();

// webassembly runner worker pool
import { useWorkerPool } from "@/stores/workerpool";
const pool = useWorkerPool();

// connection to the broker
import { useConnection } from "@/stores/connection";
const conn = useConnection();


// ---------- TRANSPORT ---------- //

// the webtransport URL to connect
let transportConfig = ref(conf.configpath);

async function connect() {
  try {
    await conf.fetchConfig(transportConfig.value);
    await conn.connect(conf.transport, conf.certhash);
  } catch (err) { terminal.error(String(err)); }
}

// connect automatically
if (conf.autoconnect) setTimeout(connect, 100);

async function rmrf() {
  let files = await opfs.rmrf()
  for (const file of files) {
    terminal.error(`Deleted: '${file}'`);
  };
};

async function listdir() {
  let files = await opfs.ls();
  if (files.length > 0) {
    terminal.log("OPFS directory listing:", LogType.Link);
    for (const file of files) {
      terminal.log(` /${file.name} [${filesize(file.size)}, ${file.type}]`, LogType.Link);
    };
  } else {
    terminal.log("OPFS directory is empty!", LogType.Link);
  };
};

function filesize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024**2) return `${(bytes/1024).toFixed(2)} KiB`;
  return `${(bytes/1024**2).toFixed(2)} MiB`;
}

async function killall() {
  await pool.killall();
  await new Promise(r => setTimeout(r, 100));
  await conn.transport?.close();
  terminal.log("Transport closed!", LogType.Danger);
};

async function shutdown() {
  await pool.ensure(0);
  await conn.transport?.close();
  terminal.log("Transport closed.", LogType.Warning);
}

// class bindings for the transport url field
const connectionStatus = computed(() => conn.connected
  ? { "is-success": true, "has-text-success": true }
  : { "is-danger": false,  "has-text-danger": false }
);


// ---------- WORKER POOL ---------- //

// fill the pool on launch
(async () => await pool.ensure(conf.workers))();

// add / remove / fill workers in the pool
async function addWorker() {
  try { await pool.add(); }
  catch (err) { terminal.error(err as string); };
};
async function terminateWorker() {
  try { await pool.terminate(); }
  catch (err) { terminal.error(err as string); };
};
async function fillWorkers() {
  try {
    await pool.fill();
    terminal.success(`Filled pool to capacity with ${pool.count} runners.`);
  } catch (err) { terminal.error(err as string); };
};


// ---------- OOM TESTING ---------- //

const showOOMtest = false;

// start a large number of `tsp.wasm` modules to try to trigger OOMs
async function runloadtesting(iterations: number = 1000) {
  terminal.success(`START LOCAL LOAD TESTING with ${iterations} iterations.`);

  // get the binary from OPFS and precompile a module
  const wasm = await opfs.getWasmModule("tsp.wasm");

  let t0 = performance.now(); // calculate how long it took
  let ooms = 0; // count the number of OOMs that surfaced
  let tasks: Promise<void>[] = []; // collect tasks to properly await

  // start lots of tasks asynchronously and await them all
  for (let count = 0; count < iterations; count++) {
    await new Promise<void>(async next => {
      let task = pool.exec(async worker => {
        try {
          await worker.run(String(count), wasm, ["tsp", "rand", "8"], [], undefined, undefined, true);
        } catch(err) {
          console.error("oops:", err);
          // just wait for OOM errors
          if (String(err).includes("Out of memory")) {
            ooms++;
          } else {
            terminal.error(String(err));
            throw err;
          };
        };
      }, next);
      tasks.push(task);
    });
  };
  await Promise.allSettled(tasks);

  // log the results
  let ms = (performance.now() - t0).toFixed(3);
  terminal.info(`Done in ${ms} ms. OOM'd ${ooms} times.`);
};

</script>

<template>
  <!-- worker pool controls -->
  <div class="columns">

    <!-- form input for the number of workers -->
    <div class="column">
      <label class="label has-text-grey-dark">Worker Pool</label>
      <div class="field has-addons">
        <div class="control">
          <input class="input is-info" type="number" min="0" max="16" step="1" placeholder="Number of Workers" disabled
            :value="pool.count" @input="ev => pool.ensure((ev.target as HTMLInputElement).value as unknown as number)"
            style="min-width: 100px;"><!-- hotfix for type="number" input ... no problem with type="text" -->
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="addWorker" :disabled="pool.count == pool.nmax" title="Add a WASM Runner to the Pool">+</button>
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="terminateWorker" :disabled="pool.count == 0" title="Remove a WASM Runner from the Pool">-</button>
        </div>
        <div class="control">
          <button class="button is-info" @click="fillWorkers" :disabled="pool.count == pool.nmax" title="Add WASM Runners to maximum capacity">Fill</button>
        </div>
      </div>

      <label class="label has-text-grey-dark">Origin-Private Filesystem</label>
      <div class="buttons">
        <button class="button is-family-monospace is-success" @click="listdir" title="List files in OPFS">ls</button>
        <button class="button is-family-monospace is-danger" @click="rmrf" title="Delete all files in OPFS">rm -rf</button>
      </div>

    </div>

    <!-- connection status -->
    <div class="column">

      <label class="label has-text-grey-dark">Broker Transport</label>
      <div class="field has-addons">
        <div class="control">
          <input :readonly="conn.connected" class="input" :class="connectionStatus" type="text" title="WebTransport Configuration URL" v-model="transportConfig">
        </div>
        <div class="control" v-if="!conn.connected">
          <button class="button is-success" @click="connect" title="Reconnect Transport">Connect</button>
        </div>
        <div class="control" v-if="conn.connected">
          <button class="button is-warning" @click="shutdown" title="Drain Workers and close the Transport gracefully">Close</button>
        </div>
        <div class="control" v-if="conn.connected">
          <button class="button is-danger" @click="killall" title="Kill Workers and close Transport immediately">Kill</button>
        </div>
      </div>

      <!-- <label class="label has-text-grey-dark">Flags</label>
      <label class="checkbox">
        <input type="checkbox">
        Log Wasm Runs
      </label> -->

      <div v-if="showOOMtest">
        <label class="label has-text-grey-dark">Out of Memory Testing</label>
        <button class="button is-warning" @click="() => runloadtesting()" title="Run Load testing to trigger OOM error">Eat my Memory, plz</button>
      </div>

    </div>

  </div>
</template>