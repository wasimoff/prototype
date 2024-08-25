#!/usr/bin/env -S deno run --allow-env --allow-read=./ --allow-net

import { create, isMessage } from "@bufbuild/protobuf";
import { Messenger } from "./transport/messenger.ts";
import { WebSocketTransport } from "./transport/websocket.ts";
import { ExecuteWasiArgsSchema, ExecuteWasiResult, ExecuteWasiResultSchema, FileListingArgsSchema, FileListingResult, FileListingResultSchema, FileProbeArgsSchema, FileProbeResult, FileProbeResultSchema, FileUploadArgsSchema, FileUploadResult, FileUploadResultSchema, GenericEventSchema, ProviderInfoSchema } from "./proto/messages_pb.ts";
import { digest, InMemoryStorage } from "./storage/filesystem.ts";
import { WasiWorkerPool } from "./workerpool/workerpool.ts";
import { parseArgs } from "@std/cli/parse-args";

// parse commandline arguments
const args = parseArgs(Deno.args, {
  alias: { "workers": "w", "help": "h" },
  default: {
    "workers": navigator.hardwareConcurrency,
    "url": "ws://localhost:4080/websocket/provider",
  },
  boolean: [ "help" ],
  string: [ "url" ],
});
if (args.help) {
  console.log("$", import.meta.filename?.replace(/.*\//, ""), "[--workers n] [--url <WebSocket URL>]");
  Deno.exit(0);
};

// validate the values
const brokerurl = args.url;
if (!/^wss?:\/\//.test(brokerurl)) throw "--url must be a WebSocket URL (wss?://)";
const nproc = Math.floor(Number(args.workers));
if (Number.isNaN(nproc) || nproc < 1) throw "--workers must be a positive number";

console.log("%c [[ wasimoff denoprovider ]] ", "color: red;");

// in-memory storage
const fs = new InMemoryStorage();

// setup the worker pool
const pool = new WasiWorkerPool(nproc, fs);
const n = await pool.fill();

// open broker connection
const ws = WebSocketTransport.connect(brokerurl);
const messenger = new Messenger(ws);

// say hello
messenger.sendEvent(create(GenericEventSchema, { message: "Hello, World!" }));
messenger.sendEvent(create(ProviderInfoSchema, {
  name: "denoprovider",
  useragent: `${navigator.userAgent} (${Deno.build.target})`,
  pool: { concurrency: n },
}));

// log received messages
(async () => {
  for await (const event of messenger.events) {
    console.log("Message: " + JSON.stringify(event));
  };
})();

// implement the rpc
for await (const rpc of messenger.requests) {
  rpc(async r => { switch (true) {

    // execute wasi binary
    case isMessage(r, ExecuteWasiArgsSchema): return <Promise<ExecuteWasiResult>>(async () => {
      let id = r.task !== undefined ? `${r.task.id}/${r.task.index}` : "unknown";
      if (r.binary === undefined || r.binary.binary.case === "raw" || r.binary.binary.value === undefined) {
        throw "raw binary not implemented yet";
      }
      let binary = r.binary.binary.value!;
      let run = await pool.run(id, { wasm: binary, argv: r.args, envs: r.envs, stdin: r.stdin });
      return create(ExecuteWasiResultSchema, {
        status: run.returncode,
        stdout: new TextEncoder().encode(run.stdout), // TODO: we've just decoded in run!
        stderr: new TextEncoder().encode(run.stderr),
      });
    })();

    // list files in storage
    case isMessage(r, FileListingArgsSchema): return <Promise<FileListingResult>>(async () => {
      let files = await Promise.all((await fs.lsf()).map(async file => ({
        filename: file.name,
        length: BigInt(file.size),
        // epoch: BigInt(file.lastModified),
        hash: await digest(file),
      })));
      console.info("Sent list of available files to Broker.");
      return create(FileListingResultSchema, { files });
    })();

    // probe for a specific file in OPFS
    case isMessage(r, FileProbeArgsSchema): return <Promise<FileProbeResult>>(async () => {
      let ok = await (async () => {
        // expect a normal file sans the bytes
        const { filename, hash, length } = r.stat!;
        // find the file by filename
        let file = (await fs.lsf()).find(f => f.name === filename);
        if (file === undefined) return false;
        // check the filesize
        if (file.size !== Number(length)) return false;
        // calculate the sha256 digest, if file exists
        if (hash.byteLength !== 32) throw new Error("hash length must be 32 bytes");
        let sum = await digest(file);
        if (sum.byteLength !== 32) throw new Error("sha-256 digest has an unexpected length");
        // compare the hashes
        for (let i in sum) if (sum[i] !== hash[i]) return false;
        // file exists and hashes match
        return true;
      })();
      return create(FileProbeResultSchema, { ok });
    })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(r, FileUploadArgsSchema): return <Promise<FileUploadResult>>(async () => {
      const { filename, hash, epoch } = r.stat!;
      const length = r.file.byteLength;
      console.log(`UPLOAD '${filename}', ${length} bytes`);
      await fs.store(r.file, filename);
      return create(FileUploadResultSchema, { ok: true });
    })();

    default:
      throw "not implemented yet";

  }}); // rpc-switch
}; // for-await

console.error("ERROR: rpc loop exited, connection lost?");
Deno.exit(1);