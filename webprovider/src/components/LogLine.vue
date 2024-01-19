<script setup lang="ts">

  // define props with default values
  withDefaults(defineProps<{
    ts: Date, // the timestamp for this line
    is: string, // an optional bulma color class
  }>(), {
    ts: () => new Date(),
    is: "info",
  });

  // create timestamp on instantiation time
  // const timestamp = new Date().toLocaleTimeString();

</script>

<template>
  <span class="has-text-grey">{{ ts.toLocaleTimeString() }}</span>&nbsp;<span :class="`has-text-${is}`"><slot></slot></span><br>
</template>