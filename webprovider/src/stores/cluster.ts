import { defineStore } from "pinia";
import { useProvider } from "./provider";
import { ref, watch } from "vue";
import { useTerminal } from "./terminal";
import { isMessage, Message } from "@bufbuild/protobuf";
import * as pb from "@wasimoff/proto/messages_pb";

export const useClusterState = defineStore("ClusterState", () => {

  const wasimoff = useProvider();
  const terminal = useTerminal();

  // number of providers currently connected to the broker
  const providers = ref<number>();

  // current throughput of tasks per second
  const throughput = ref<number>(0);

  // whenever the provider messenger reconnects
  watch(() => wasimoff.$messenger, async (messenger) => {
    if (messenger !== undefined && wasimoff.$provider !== undefined) {

      // transfer a readablestream from the provider directly
      // const stream = await wasimoff.$provider.getEventstream();

      // get the event iterator's next function and create a stream ourselves
      const next = await wasimoff.$provider.getEventIteratorNext();
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
          case isMessage(event, pb.GenericEventSchema):
            terminal.info(`Message: ${event.message}`);
            break;

          // update provider count
          case isMessage(event, pb.ClusterInfoSchema):
            providers.value = event.providers;
            break;

          // update throughput
          case isMessage(event, pb.ThroughputSchema):
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
