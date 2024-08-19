import { ref, shallowRef, watch, type ComputedRef, computed } from "vue";
import { defineStore, storeToRefs } from "pinia";
import { useWorkerPool } from "@/stores/workerpool";
import { Messenger, WebSocketTransport } from "@/transports";
import { useTerminal } from "./terminal";
import { OPFSDirectory } from "@/filesystem/opfs";
import type { WasiTaskExecution, WasiTaskResult } from "@/workerpool/wasiworker";
import { Trace } from "@/fn/trace";
import { proxy } from "comlink";
import { EventSchema, FileStatSchema, RequestSchema, ResponseSchema, type Event, type FileStat, type Request, type Response } from "@/proto/messages_pb";
import { create, type MessageInitShape } from "@bufbuild/protobuf";

// TODO, check out these additions / alternatives:
// - CompressionStream could reduce the size on the wire
// - experimental WebSocketStream looks interesting, https://developer.chrome.com/articles/websocketstream/


/** The connection store abstracts away the details of the underlying transport. */
export const useConnection = defineStore("Connection", () => {
  const prefix = [ "%c Connection ", "background: goldenrod; color: black;" ];

  // use other stores for terminal output and filesystem access
  const terminal = useTerminal();
  let filesystem: OPFSDirectory;
  (async () => filesystem = await OPFSDirectory.open("/wasm"))();

  // use the worker pool needed to execute WASM
  let pool = useWorkerPool();

  // keep reference to the connection itself
  const transport = shallowRef<Messenger | null>(null);
  const connected = computed(() => transport.value !== null);

  // send events
  async function send(event: Event) {
    if (transport.value === null) throw "transport is null";
    return transport.value.sendEvent(event);
  };

  async function connect(url: string, options: any) {

    // close any previous connections
    // TODO: close iterators?
    if (transport.value) transport.value.close();
    transport.value = null;

    // connect the new transport
    console.log(...prefix, "to", url);
    let wst = WebSocketTransport.connect(url);
    let messenger = new Messenger(wst);
    transport.value = messenger;
    console.log(...prefix, "established", messenger);
    terminal.info("WebTransport connection established.");

    // handle connection failures
    let toStr = (o: any) => (typeof o === "object") ? JSON.stringify(o) : String(o);
    messenger.closed.addEventListener("abort", (reason) => {
      transport.value = null;
      console.log(...prefix, `closed:`, reason);
      terminal.error(`Broker connection closed!`);
    });

    // --------- MESSAGES ---------

    // send initial information about this provider
    providerInfo(); poolInfo();
    // watch the pool size and send updates
    watch(storeToRefs(pool).count, async () => poolInfo());

    // log received messages
    (async () => {
      for await (let event of messenger.events) {
        terminal.info("Message: " + JSON.stringify(event.event));
      }
      console.error(...prefix, "Event stream ended.");
    })();

    // --------- RPC REQUESTS ---------
    // handle rpc requests
    (async () => {
      // for each incoming request ...
      for await (let rpc of messenger.requests) {

        // This Promise is resolved with `next` when this loop shall continue to the next
        // request in the iterable. It was originally meant as a backpressure mechanism.
        // Currently, the only WebSocket transport does not support backpressure though,
        // so it's a NOP here.
        await new Promise<void>(async next => {
          next();
          await rpc(rpchandler);
        });

      }
      // the iteration over the async generator somehow stopped
      console.log(...prefix, "RPC Stream has ended");
    })();

  };

  /** Send information about this Provider to the Broker. */
  async function providerInfo() {
    return send(create(EventSchema, { event: {
      case: "providerInfo",
      value: {
        name: "unknown webprovider",
        platform: navigator.platform,
        useragent: navigator.userAgent,
      },
    }}));
  };

  /** Send updates on worker pool capacity to the Broker. */
  async function poolInfo() {
    return send(create(EventSchema, { event: {
      case: "providerResources",
      value: {
        concurrency: pool.count,
        // tasks: pool.count,
      }
    }}));
  };


  async function rpchandler(request: Request): Promise<Response> {
    let r = request.request;
    switch (r.case) {

      case "executeWasiArgs": return (async () => {
        let v = r.value;
        let id = v.task !== undefined ? `${v.task.id}/${v.task.index}` : "unknown";
        if (v.binary === undefined || v.binary.binary.case === "raw" || v.binary.binary.value === undefined) {
          return create(ResponseSchema, { error: `raw binary not implemented yet` });
        }
        let binary = v.binary.binary.value!;
        // maybe start a trace
        let tracer: Trace | undefined;
        if (v.trace === true) tracer = new Trace("rpc: function top");
        // assembly advanced options
        // let options: AdvancedExecutionOptions = {};
        // // preload files under exactly their names in OPFS storage, as a simplification
        // if (loadfs) options.rootfs = loadfs.reduce((o,v) => { o[v] = v; return o; }, { } as { [k: string]: string });
        // if (datafile) options.datafile = datafile;
        // if (stdin) options.stdin = stdin;
        tracer?.now("rpc: parsed options");
        let run = await pool.run(id, { wasm: binary, argv: v.args, envs: v.envs, stdin: v.stdin });
        return create(ResponseSchema, { response: {
          case: "executeWasiResult",
          value: {
            status: run.returncode,
            stdout: new TextEncoder().encode(run.stdout), // TODO: we've just decoded in run!
            stderr: new TextEncoder().encode(run.stderr),
          },
        }});
        // return await pool.exec(async worker => {
        //   tracer?.now("rpc: pool.exec got a worker");
        //   return await worker.run(id, binary, [binary, ...args], envs, options, trace ? proxy(tracer!) : undefined);
        // }, next);
      })();

      case "executeWasmArgs": // TODO: plain wasm
        return create(ResponseSchema, { error: `not implemented yet: ${request.request.case}` });

      // list files in OPFS
      case "fileListingArgs": return (async () => {
        let files = await Promise.all((await filesystem.lsf()).map(async file => ({
          filename: file.name,
          contenttype: file.type,
          length: BigInt(file.size),
          epoch: BigInt(file.lastModified),
          hash: await digest(file),
        })));
        terminal.info("Sent list of available files to Broker.");
        return create(ResponseSchema, { response: {
          case: "fileListingResult",
          value: { files },
        }});
      })();

      // probe for a specific file in OPFS
      case "fileProbeArgs": return (async () => {
        let ok = await (async () => {
          // expect a normal file sans the bytes
          const { filename, hash, length } = r.value.file!;
          // find the file by filename
          let file = (await filesystem.lsf()).find(f => f.name === filename);
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
        return create(ResponseSchema, { response: {
          case: "fileProbeResult",
          value: { ok },
        }});
      })();

      // binaries uploaded from the broker inside an rpc
      case "fileUploadArgs": return (async () => {
        const { filename, hash, epoch } = r.value.stat!;
        const length = r.value.file.byteLength;
        console.log(...prefix, `UPLOAD '${filename}', ${length} bytes`);
        await filesystem.store(r.value.file, filename);
        terminal.success(`Uploaded new file: '${filename}', ${length} bytes`);
        return create(ResponseSchema, { response: {
          case: "fileUploadResult",
          value: { ok: true },
        }});
      })();

    };
    // everything else is an error / not implemented yet
    return create(ResponseSchema, { error: `not implemented yet: ${request.request.case}` });
  }

  return { transport, connected, connect };
});

// digest a file to a [32]Uint8Array
async function digest(file: File, verbose = false): Promise<Uint8Array> {
  let t0 = performance.now();
  let sum = new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()));
  if (verbose) console.warn("SHA-256 digest of", file.name, `(${file.size} bytes)`, "took", performance.now() - t0, "ms.");
  return sum;
}
