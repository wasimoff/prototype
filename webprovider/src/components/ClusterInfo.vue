<script setup lang="ts">

import { storeToRefs } from "pinia";
import { useClusterState } from "@app/stores/cluster";

const cluster = useClusterState();

const { providers, throughput } = storeToRefs(cluster);

</script>

<template>

  <!-- show number of providers at broker -->
  <p v-if="providers !== undefined">
    Broker has {{ providers }} connected provider{{ providers > 1 ? "s" : "" }}.
  </p>
  <p v-else>
    You're not connected to any Broker.
  </p>

  <!-- show current throughput -->
   <p>
    Current throughput: {{ throughput }} tasks/sec
   </p>

</template>