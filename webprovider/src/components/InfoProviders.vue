<script setup lang="ts">

import { storeToRefs } from "pinia";
import { useClusterState } from "@app/stores/cluster";

const cluster = useClusterState();

const { providers } = storeToRefs(cluster);

</script>

<template>
  <span v-if="providers !== undefined">
    Currently {{ providers }} provider{{ providers > 1 ? "s" : "" }} connected.
  </span>
</template>