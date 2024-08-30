#!/usr/bin/env -S deno run --allow-env --allow-read=./,../webprovider/ --allow-net

import { create, isMessage } from "@bufbuild/protobuf";
import { GenericEventSchema } from "@wasimoff/proto/messages_pb.ts";
import { parseArgs } from "@std/cli/parse-args";
import { WasimoffProvider } from "@wasimoff/worker/provider.ts";

// parse commandline arguments
const help = (fatal: boolean = false) => {
  console.log("$", import.meta.filename?.replace(/.*\//, ""), "[--workers n] [--url <WebSocket URL>]");
  Deno.exit(fatal ? 1 : 0);
};
const args = parseArgs(Deno.args, {
  alias: { "workers": "w", "url": "u", "help": "h" },
  default: {
    "workers": navigator.hardwareConcurrency,
    "url": "ws://localhost:4080/websocket/provider",
  },
  boolean: [ "help" ],
  string: [ "url" ],
  unknown: (arg) => { console.warn("Unknown argument:", arg); help(true); }
});

// print help if requested
if (args.help) help();

// validate the values
const brokerurl = args.url;
if (!/^wss?:\/\//.test(brokerurl)) throw "--url must be a WebSocket URL (wss?://)";
const nproc = Math.floor(Number(args.workers));
if (Number.isNaN(nproc) || nproc < 1) throw "--workers must be a positive number";

// initialize the provider
console.log("%c [[ Wasimoff Denoprovider ]] ", "color: red;");
const provider = await WasimoffProvider.init(nproc, brokerurl, ":memory:");
const workers = await provider.pool.fill();
await provider.sendInfo(workers, "deno", `${navigator.userAgent} (${Deno.build.target})`);

// say hello
provider.messenger?.sendEvent(create(GenericEventSchema, { message: "Hello, World!" }));

// log received messages
(async () => {
  for await (const event of provider.messenger.events) {
    if (isMessage(event, GenericEventSchema))
      console.log("Message: " + JSON.stringify(event));
  };
})();

// start handling requests
await provider.handlerequests();

console.error("ERROR: rpc loop exited, connection lost?");
Deno.exit(1);
