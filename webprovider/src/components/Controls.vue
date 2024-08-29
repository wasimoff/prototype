<script setup lang="ts">
import { computed, watch } from "vue";
import { storeToRefs } from "pinia";

// terminal for logging
import { useTerminal, LogType } from "@app/stores/terminal.ts";
const terminal = useTerminal();

// configuration via url fragment
import { useConfiguration } from "@app/stores/configuration.ts";
const conf = useConfiguration();

// the broker socket to connect
const transport = storeToRefs(conf).transport;

// link to the provider worker
import { useProvider } from "@app/stores/provider.ts";
const wasimoff = useProvider();
// TODO: typings for ref<remote<...> | undefined>?
const { connected, workers, $pool, $provider, $storage } = storeToRefs(wasimoff);

// connect immediately on load, when the provider proxy is connected
let stop = watch(() => wasimoff.$provider, async (provider) => {
  if (provider !== undefined) {
    stop(); // do this just once

    // TODO: connect to configuration store
    await wasimoff.open(":memory:");
    terminal.log(`Opened in-memory storage.`, LogType.Info);

    // add at least one worker immediately
    if (workers.value === 0) await $pool.value?.scale(1);

    // connect to the broker
    await connect();

    // fill remaining workers to capacity
    if ($pool.value) {
      await fillWorkers();
      // while (await $pool.value.spawn());
      // terminal.log(`Provider filled with ${workers.value} Workers.`, LogType.Info);
    };

  };
});


async function connect() {
  try {
    const url = transport.value;
    await wasimoff.connect(url);
    terminal.log(`Connected to Broker at ${url}`, LogType.Success);
    await $provider.value?.sendInfo(workers.value);
    wasimoff.handlerequests();
  } catch (err) { terminal.error(String(err)); }
}

// connect automatically
// if (conf.autoconnect) setTimeout(connect, 100);

async function rmrf() {
  if (!$storage.value) return terminal.error("$storage not connected yet");
  let files = await $storage.value.prune();
  for (const file of files) {
    terminal.error(`Deleted: '${file}'`);
  };
};

async function listdir() {
  if (!$storage.value) return terminal.error("$storage not connected yet");
  let items = await $storage.value.lsf();
  if (items.length > 0) {
    terminal.log(`Directory listing:`, LogType.Link);
    function filesize(bytes: number): string {
      if (bytes < 1024) return `${bytes} B`;
      if (bytes < 1024**2) return `${(bytes/1024).toFixed(2)} KiB`;
      return `${(bytes/1024**2).toFixed(2)} MiB`;
    };
    for (const it of items) {
      if (it instanceof File)
        terminal.log(` ${it.name} [${filesize(it.size)}, ${it.type}]`, LogType.Link);
      // else
      //   terminal.log(` ${it.name}/ [directory]`, LogType.Link);
    };
  } else {
    terminal.log(`Directory is empty!`, LogType.Link);
  };
};

async function kill() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  await $pool.value.killall();
  // grace period for some error responses
  await new Promise(r => setTimeout(r, 100));
  await wasimoff.disconnect();
};

async function shutdown() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  await $pool.value.scale(0);
  await wasimoff.disconnect();
}

// class bindings for the transport url field
const connectionStatus = computed(() => connected.value
  ? { "is-success": true, "has-text-success": true }
  : { "is-danger": false,  "has-text-danger": false }
);

// watch connection status disconnections
watch(() => connected.value, (conn) => {
  if (conn === false) terminal.log("Connection closed.", LogType.Warning);
});


// ---------- WORKER POOL ---------- //

// add / remove / fill workers in the pool
async function spawnWorker() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try { await $pool.value.spawn(); }
  catch (err) { terminal.error(err as string); };
};
async function dropWorker() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try { await $pool.value.drop(); }
  catch (err) { terminal.error(err as string); };
};
async function scaleWorker(n?: number) {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try { await $pool.value.scale(n); }
  catch (err) { terminal.error(err as string); };
};
async function fillWorkers() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try {
    // await $pool.value.fill();
    let max = await $pool.value.capacity;
    while (await $pool.value.spawn() < max);
    terminal.success(`Filled pool to capacity with ${workers.value} runners.`);
  } catch (err) { terminal.error(err as string); };
};

// TODO: forward from pool
const nmax = 16

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
            :value="workers" @input="ev => scaleWorker((ev.target as HTMLInputElement).value as unknown as number)"
            style="min-width: 100px;"><!-- hotfix for type="number" input ... no problem with type="text" -->
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="spawnWorker" :disabled="workers == nmax" title="Add a WASM Runner to the Pool">+</button>
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="dropWorker" :disabled="workers == 0" title="Remove a WASM Runner from the Pool">-</button>
        </div>
        <div class="control">
          <button class="button is-info" @click="fillWorkers" :disabled="workers == nmax" title="Add WASM Runners to maximum capacity">Fill</button>
        </div>
      </div>

      <label class="label has-text-grey-dark">Storage</label>
      <div class="buttons">
        <button class="button is-family-monospace is-success" @click="listdir" title="List files in OPFS">ls</button>
        <button class="button is-family-monospace is-danger" @click="rmrf" title="Delete all files in OPFS">rm -f *</button>
      </div>

    </div>

    <!-- connection status -->
    <div class="column">

      <label class="label has-text-grey-dark">Broker Transport</label>
      <div class="field has-addons">
        <div class="control">
          <input :readonly="connected" class="input" :class="connectionStatus" type="text" title="WebTransport Configuration URL" v-model="transport">
        </div>
        <div class="control" v-if="!connected">
          <button class="button is-success" @click="connect" title="Reconnect Transport">Connect</button>
        </div>
        <div class="control" v-if="connected">
          <button class="button is-warning" @click="shutdown" title="Drain Workers and close the Transport gracefully">Close</button>
        </div>
        <div class="control" v-if="connected">
          <button class="button is-danger" @click="kill" title="Kill Workers and close Transport immediately">Kill</button>
        </div>
      </div>

    </div>

  </div>
</template>