import { ref, shallowRef, watch, type ComputedRef, computed } from "vue";
import { defineStore, storeToRefs } from "pinia";
import { useWorkerPool } from "@/stores/workerpool";
import { WebTransportBroker, BrokerTransport } from "@/transports";
import { useTerminal } from "./terminal";
import { OPFSDirectory } from "@/filesystem/opfs";
import type { WasiTaskExecution, WasiTaskResult } from "@/workerpool/wasiworker";
import { Trace } from "@/fn/trace";
import { proxy } from "comlink";

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
  const transport = shallowRef<BrokerTransport | null>(null);
  const connected = computed(() => transport.value !== null);

  // send control messages over the transport
  async function send<T>(type: string, message: T) {
    if (transport.value === null) throw "transport is not connected";
    return transport.value.messages.send({ type, message });
  };

  // access rpc and message channels when transport is connected
  const rpc = computed(() => {
    if (!connected.value) throw "transport is not connected";
    return transport.value!.rpc;
  });
  const messages = computed(() => {
    if (!connected.value) throw "transport is not connected";
    return transport.value!.messages;
  });

  async function connect(url: string, certhash?: string) {

    // close any previous connections
    if (transport.value) transport.value.close();
    transport.value = null;

    // connect the new transport
    console.log(...prefix, "to WebTransport", url);
    let broker = await WebTransportBroker.connect(url, certhash);
    transport.value = broker;
    console.log(...prefix, "established", broker);
    terminal.info("WebTransport connection established.");

    // handle connection failures
    let toStr = (o: any) => (typeof o === "object") ? JSON.stringify(o) : String(o);
    broker.closed
      .then(info => {
        transport.value = null;
        console.log(...prefix, `closed gracefully:`, info);
        terminal.info(`Broker connection closed gracefully: ` + toStr(info));
      })
      .catch(err => {
        transport.value = null;
        console.error(...prefix, `closed unexpectedly:`, err);
        terminal.error(`Broker connection closed unexpectedly: ` + toStr(err));
      });

    // --------- MESSAGES ---------
    // send a "greeting" control message
    await broker.messages.send({ type: "hello", message: {
      sent: new Date().toISOString(),
    }});

    // send information about this provider
    providerInfo(); poolInfo();
    watch(storeToRefs(pool).count, async () => poolInfo());

    // log received messages
    (async () => {
        try {
        for await (const message of broker.messages.channel) {
          terminal.info("Message: " + JSON.stringify(message));
        };
      } catch (err) {
        console.error(...prefix, "Message Stream failed:", err);
        // terminal.error("Message Stream failed: " + String(err));
      };
    })();

    // --------- RPC REQUESTS ---------
    // handle rpc requests
    (async () => {
      try {
        for await (const request of broker.rpc) {
          // `next` is resolved when we should continue to the next iteration
          await new Promise<void>(async next => {
            try {
              // try to handle the request with a method from the rpc map
              await request(async (method, body) => {
                if (method in methods) return await methods[method](body, next);
                else throw "unknown method";
              });
            } finally {
              // make sure to always resolve the Promise, if the handler didn't call it
              next();
            };
          });
        };
        // the iteration over the async generator somehow stopped
        console.log(...prefix, "RPC Stream has ended");
      } catch(err) {
        console.error(...prefix, "RPC Handler failed:", err);
        broker.close();
      };
    })();

  };

  /** Send information about this Provider to the Broker. */
  async function providerInfo() {
    return send<ProviderInfo>("providerinfo", {
      platform: navigator.platform,
      useragent: navigator.userAgent,
    });
  };

  /** Send updates on worker pool capacity to the Broker. */
  async function poolInfo() {
    return send<PoolInfo>("poolinfo", {
      nmax: pool.nmax,
      pool: pool.count,
    });
  };


  /** This hashtable contains all the known RPC function that may be called by the client broker.
   * * the key is the `method` name
   * * the RPC arguments are passed via `body`
   * * and `next` shall be called when the higher RPC request loop shall continue its next iteration.
   **/
  const methods: { [method: string]: (body: any, next: () => void) => Promise<any> } = {

    /** Execute a stored WASI executable via the /run endpoint in Broker. */
    async run(body, next): Promise<WasiTaskResult> {
      // expected body type
      let { id, binary, args, envs, stdin, loadfs, datafile, trace } = body as WASMRun;
      // maybe start a trace
      let tracer: Trace | undefined;
      if (trace === true) tracer = new Trace("rpc: function top");
      // undefined slices get encoded as `null` in Go
      if (args === null) args = [];
      if (envs === null) envs = [];
      if (loadfs === null) loadfs = [];
      // assembly advanced options
      // let options: AdvancedExecutionOptions = {};
      // // preload files under exactly their names in OPFS storage, as a simplification
      // if (loadfs) options.rootfs = loadfs.reduce((o,v) => { o[v] = v; return o; }, { } as { [k: string]: string });
      // if (datafile) options.datafile = datafile;
      // if (stdin) options.stdin = stdin;
      tracer?.now("rpc: parsed options");
      let run = pool.run(id, { wasm: binary, argv: args, envs, stdin });
      next();
      return run;
      // return await pool.exec(async worker => {
      //   tracer?.now("rpc: pool.exec got a worker");
      //   return await worker.run(id, binary, [binary, ...args], envs, options, trace ? proxy(tracer!) : undefined);
      // }, next);
    },

    /** Broker probes if we have a certain file. */
    async "fs:probe"(body: UploadedFile, next): Promise<boolean> {
      // expect a normal file sans the bytes
      const { filename, hash, length } = body;
      // find the file by filename
      let file = (await filesystem.lsf()).find(f => f.name === filename);
      if (file === undefined) return false;
      // check the filesize
      if (file.size !== length) return false;
      // calculate the sha256 digest, if file exists
      if (hash.byteLength !== 32) throw new Error("hash length must be 32 bytes");
      let sum = await digest(file);
      if (sum.byteLength !== 32) throw new Error("sha-256 digest has an unexpected length");
      // compare the hashes
      for (let i in sum) if (sum[i] !== hash[i]) return false;
      // file exists and hashes match
      return true;
    },

    /** Binaries "uploaded" from the Broker via RPC. */
    async "fs:upload"(body: UploadedFile, next): Promise<boolean> {
      // expect a filename and a binary
      const { filename, bytes, hash, epoch } = body;
      console.log(...prefix, `UPLOAD '${filename}', ${bytes.byteLength} bytes`);
      await filesystem.store(bytes, filename);
      terminal.success(`Uploaded new file: '${filename}', ${bytes.byteLength} bytes`);
      return true;
    },

    /** Broker asks for a list of available files in storage. */
    async "fs:list"(body: null, next): Promise<Partial<UploadedFile>[]> {
      let has = await Promise.all((await filesystem.lsf()).map(async file => ({
        filename: file.name,
        hash: await digest(file),
        length: file.size,
        epoch: BigInt(file.lastModified),
      })));
      terminal.info("Sent list of available files to Broker.");
      return has;
    },

    /** Simple Ping-Pong message to say hello. */
    async ping(body: string, next): Promise<string> {
      if (body != "ping") throw "expected a 'ping'";
      return "pong";
    },

  }; /* methods */

  return { transport, connected, connect };
});

// expected body type for a WASM run configuration
// see provider.go WASMRun struct
type WASMRun = {
  id: string,
  binary: string,
  args: string[],
  envs: string[],
  stdin?: string,
  loadfs?: string[],
  datafile?: string,
  trace: boolean,
}

// expected body type for file uploads
type UploadedFile = {
  filename:   string,
  bytes:      Uint8Array,
  hash:       Uint8Array,
  length:     number,
  epoch:      BigInt,
}

// digest a file to a [32]Uint8Array
async function digest(file: File, verbose = false): Promise<Uint8Array> {
  let t0 = performance.now();
  let sum = new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()));
  if (verbose) console.warn("SHA-256 digest of", file.name, `(${file.size} bytes)`, "took", performance.now() - t0, "ms.");
  return sum;
}

// information about this provider to send to the broker
type ProviderInfo = {
  name?: string;
  platform: string;
  useragent: string;
};

// updates about the pool capacity
type PoolInfo = {
  nmax: number,
  pool: number,
}