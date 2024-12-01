<script setup lang="ts">
import { computed, watch, ref } from "vue";
import { storeToRefs } from "pinia";

// terminal for logging
import { useTerminal, LogType } from "@app/stores/terminal.ts";
const terminal = useTerminal();

// configuration via url fragment
import { useConfiguration } from "@app/stores/configuration.ts";
const conf = useConfiguration();

// the broker socket to connect
const transport = ref(conf.transport);

// info components
import InfoProviders from "./ClusterInfo.vue";

// link to the provider worker
import { useProvider } from "@app/stores/provider.ts";
let wasimoff = useProvider();
// TODO: typings for ref<remote<...> | undefined>?
const { connected, workers, $pool } = storeToRefs(wasimoff);

// connect immediately on load, when the provider proxy is connected
watch(() => wasimoff.$provider, async (provider) => {
  if (provider !== undefined) {

    await wasimoff.open(":memory:", transport.value);
    // terminal.success(`Opened in-memory storage.`);

    // add at least one worker immediately
    if (workers.value.length === 0) await $pool.value?.scale(1);

    // maybe autoconnect to the broker
    if (conf.autoconnect) await connect();
    else terminal.warn("Autoconnect disabled. Please connect manually.");

    // fill remaining workers to capacity
    if ($pool.value) await fillWorkers();

  };
});

async function connect() {
  try {
    const url = transport.value;
    await wasimoff.connect(url);
    terminal.log(`Connected to Broker at ${url}`, LogType.Success);
    wasimoff.handlerequests();
  } catch (err) { terminal.error(String(err)); }
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
  ? { "is-success": true, "has-text-grey": true }
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
    terminal.success(`Filled pool to capacity with ${workers.value.length} runners.`);
  } catch (err) { terminal.error(err as string); };
};

// get the maximum capacity from pool
const nmax = ref(0);
watch(() => wasimoff.$pool, async value => {
  // capacity is readonly and should only ever change once
  if (value) nmax.value = await value.capacity;
  stop();
});

// calculate current worker usage
const usage = computed(() => {
  const busy = workers.value;
  if (busy.length === 0) return 0;
  return busy.filter(b => b).length / busy.length;
});

</script>

<template>

  <!-- worker pool controls -->
  <div class="columns">

    <!-- form input for the number of workers -->
    <div class="column">
      <label class="label has-text-grey-dark">Worker Pool</label>
      <div class="field has-addons">
        <div class="control">
          <input class="input is-info" type="text" placeholder="Number of Workers" disabled
            :value="workers.length" @input="ev => scaleWorker((ev.target as HTMLInputElement).value as unknown as number)"
            style="width: 110px;"><!-- hotfix for type="number" input ... no problem with type="text" -->
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="spawnWorker" :disabled="workers.length == nmax" title="Add a WASM Runner to the Pool">+</button>
        </div>
        <div class="control">
          <button class="button is-family-monospace is-info" @click="dropWorker" :disabled="workers.length == 0" title="Remove a WASM Runner from the Pool">-</button>
        </div>
        <!-- hidden for demo -->
        <!-- <div class="control">
          <button class="button is-info" @click="fillWorkers" :disabled="workers.length == nmax" title="Add WASM Runners to maximum capacity">Fill</button>
        </div> -->
      </div>

      <label class="label has-text-grey-dark">Current Usage: {{ (100*usage).toFixed() }}%</label>
      <progress class="progress is-large is-info" :value="usage" max="1" style="width: 245px;"></progress>

    </div>

    <!-- connection status -->
    <div class="column">

      <label class="label has-text-grey-dark">Broker Transport</label>
      <div class="field has-addons">
        <div class="control">
          <input :readonly="connected" class="input" :class="connectionStatus" type="text" title="Broker URL" v-model="transport">
        </div>
        <div class="control" v-if="!connected">
          <button class="button is-success" @click="connect" title="Connect Transport">Connect</button>
        </div>
        <div class="control" v-if="connected">
          <button class="button is-warning" @click="shutdown" title="Drain Workers and close the Transport gracefully">Disconnect</button>
        </div>
        <!-- hidden for demo -->
        <!-- <div class="control" v-if="connected">
          <button class="button is-danger" @click="kill" title="Kill Workers and close Transport immediately">Kill</button>
        </div> -->
      </div>

      <label class="label has-text-grey-dark">Cluster Information</label>
      <InfoProviders></InfoProviders>

    </div>

  </div>
</template>
