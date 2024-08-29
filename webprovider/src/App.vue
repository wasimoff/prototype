<script setup lang="ts">

import Terminal from "@app/components/Terminal.vue";
import Controls from "@app/components/Controls.vue";

import { useTerminal, LogType } from "@app/stores/terminal.ts";
const terminal = useTerminal();
terminal.log("Starting wasimoff provider ...", LogType.Black);

// check if we are in a secure context
if (!window.isSecureContext) {
  terminal.warn("WARNING: application running in an insecure context!")
}

</script>

<template>

  <!-- logo and title -->
  <h1 class="title">
    <img alt="WebAssembly Logo" class="logo" src="./assets/wasm.svg" />
    wasimoff
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
