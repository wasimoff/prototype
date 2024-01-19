<script setup lang="ts">

import Terminal from "@/components/Terminal.vue";
import Controls from "@/components/Controls.vue";

const title = "wasimoff";

import { useTerminal, LogType } from "@/stores/terminal";
const terminal = useTerminal();
terminal.log(`Hello, ${title}!`, LogType.Black);

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
