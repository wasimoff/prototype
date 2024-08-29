<script setup lang="ts">

import { ref, watch } from "vue";
import { useTerminal } from "@app/stores/terminal";
import LogLine from "@app/components/LogLine.vue";

// reference to the log textarea
const logarea = ref<HTMLSpanElement | null>(null);

// use the store to display the logged lines
const terminal = useTerminal();

// scroll the textarea on changes to the log lines
watch(terminal.lines, scrolldown, { deep: false, flush: "post" });
function scrolldown() {
  const area = logarea.value;
  if (!!area) area.scrollTop = area.scrollHeight;
};

</script>

<template>
  <span class="textarea is-family-monospace" ref="logarea">
    <LogLine v-for="line in terminal.lines" :ts="line.date" :is="line.is">{{ line.text }}</LogLine>
    <span v-if="!terminal.lines.length" class="has-text-grey-lighter">Log Messages will appear here</span>
  </span>
</template>

<style scoped>
span.textarea {
  /* scroll-behavior: smooth; */
  scrollbar-width: none;
  height: 500px;
  overflow-y: scroll;
  white-space: pre-wrap;
}
</style>
