<script setup lang="ts">

import { storeToRefs } from "pinia";
import { useClusterState } from "@app/stores/cluster";

const cluster = useClusterState();

const { providers } = storeToRefs(cluster);

</script>

<template>
  <span v-if="providers !== undefined">
    Broker has {{ providers }} connected provider{{ providers > 1 ? "s" : "" }}.
  </span>
  <span v-else>
    You're not connected to any Broker.
  </span>
</template>