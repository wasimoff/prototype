import { defineStore } from "pinia";
import { useProvider } from "./provider";
import { ref, watch } from "vue";
import { useTerminal } from "./terminal";
import { isMessage } from "@bufbuild/protobuf";
import * as pb from "@wasimoff/proto/messages_pb";

export const useClusterState = defineStore("ClusterState", () => {

  const wasimoff = useProvider();
  const terminal = useTerminal();

  // number of providers currently connected to the broker
  const providers = ref<number>();

  // whenever the provider messenger reconnects
  watch(() => wasimoff.$messenger, async (messenger) => {
    if (messenger !== undefined && wasimoff.$provider !== undefined) {

      // read messages from the event stream
      for await (const event of await wasimoff.$provider.getEventstream()) {
        switch (true) { // switch by message type

          // print generic messages to the terminal
          case isMessage(event, pb.GenericEventSchema):
            terminal.info(`Message: ${event.message}`);
            break;

          // update provider count
          case isMessage(event, pb.ClusterInfoSchema):
            providers.value = event.providers;
            break;

        };
      };

      // tidy up when disconnected
      providers.value = undefined;

    };
  });


  return {
    providers,
  };

})