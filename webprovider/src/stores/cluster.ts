import { defineStore } from "pinia";
import { useProvider } from "./provider";
import { ref, watch } from "vue";
import { useTerminal } from "./terminal";
import { isMessage, Message } from "@bufbuild/protobuf";
import * as wasimoff from "@wasimoff/proto/v1/messages_pb";

export const useClusterState = defineStore("ClusterState", () => {

  const providerstore = useProvider();
  const terminal = useTerminal();

  // number of providers currently connected to the broker
  const providers = ref<number>();

  // current throughput of tasks per second
  const throughput = ref<number>(0);

  // whenever the provider messenger reconnects
  watch(() => providerstore.$messenger, async (messenger) => {
    if (messenger !== undefined && providerstore.$provider !== undefined) {

      // transfer a readablestream from the provider directly
      // const stream = await wasimoff.$provider.getEventstream();

      // get the event iterator's next function and create a stream ourselves
      const next = await providerstore.$provider.getEventIteratorNext();
      const stream = new ReadableStream<Message>({
        async pull(controller) {
          let { done, value } = await next();
          if (done) return controller.close();
          if (value) controller.enqueue(value);
        },
      });

      // read messages from the event stream
      for await (const event of stream) {
        switch (true) { // switch by message type

          // print generic messages to the terminal
          case isMessage(event, wasimoff.Event_GenericMessageSchema):
            terminal.info(`Message: ${event.message}`);
            break;

          // update provider count
          case isMessage(event, wasimoff.Event_ClusterInfoSchema):
            providers.value = event.providers;
            break;

          // update throughput
          case isMessage(event, wasimoff.Event_ThroughputSchema):
            throughput.value = event.overall;
            break;

        };
      };

      // tidy up when disconnected
      providers.value = undefined;

    };
  });


  return { providers, throughput };

})
