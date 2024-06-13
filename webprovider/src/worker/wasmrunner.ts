// A Worker which runs WebAssembly modules with a WASI shim.
// Binaries are either transferred through the Comlink or are read from OPFS.
/// <reference lib="webworker" />

import { OpenFile, File, WASI, PreopenDirectory, SyncOPFSFile, Fd, strace, Directory } from "@bjorn3/browser_wasi_shim";
import { expose, workerReady, type ProxyMarked } from "@/workerpool";
import type { useTerminal } from "@/stores/terminal";
import type { useFilesystem } from "@/stores/filesystem";
import type { Trace, ExportedTrace } from "@/fn/trace";

// expect to instantiate WASI modules
type WasiInstance = { exports: { memory: WebAssembly.Memory, _start: () => any } };

// a preloaded root filesystem
type RootFS = { [key: string]: File | SyncOPFSFile | Directory };


/** Advanced options for a task execution. */
export type AdvancedExecutionOptions = {

  /** Put a string on `stdin`, instead of an empty file. */
  stdin?: string,

  /** Construct the root filesystem by loading files from OPFS.
   * The keys are filenames in the WASI context's filesystem,
   * the value is a filename, which is loaded from OPFS as the file's contents. */
  // TODO: use .tar.gz unpacker for full "images" here
  rootfs?: { [filename: string]: string },

  /** Select a specific data file in the filesystem to return to caller,
   * along with `stdout` and `stderr`. */
  datafile?: string,

  /** Wrap the WASI imports in `strace` for improved debug visibility. */
  strace?: boolean,

};


/** Struct that is returned by a task execution. */
export type CompletedExecution = {
  exitcode: number,         // the exit code, where 0 is usually "success"
  stdout: string,           // standard output, decoded as string
  stderr: string,           // standard error, decoded as string
  datafile?: ArrayBuffer,   // the requested datafile contents from filesystem
  trace?: ExportedTrace,    // a trace of events with microsecond unix epochs
};


export class WASMRunner {

  //! FinalizationRegistry test to get notified when WebAssembly.Memory is collected
  // @ts-ignore
  private registry = new FinalizationRegistry(m => { if (this.verbose) console.warn(...this.prefix, m); });

  // colorful console logging prefix
  private get prefix() { return [ `%c ${this.name} `, 'background: #f03a5f; color: white;' ]; }
  public name: string;

  // flag indicating whether to print lots of info to the console
  //! makes overall execution quite a bit slower
  public verbose: boolean;

  /** Instantiate a new `WASMRunner` with a `name` and a Comlink-proxy to the `terminal` for logging. */
  constructor(
    name: string,
    private terminal: ReturnType<typeof useTerminal> & ProxyMarked,
    private filesystem: ReturnType<typeof useFilesystem> & ProxyMarked,
    verbose: boolean = false,
  ) {
    this.name = `WASMRunner:${name}`;
    this.verbose = verbose;
  };

  /** Preload the WASI context with files from OPFS. */
  private async preloadRootfs(fs: RootFS, load?: AdvancedExecutionOptions["rootfs"]) {
    if (load) {
      for (let [key, file] of Object.entries(load)) {
        try {
          // load the file from OPFS
          let contents = await this.filesystem.getCached(file);
          fs[key] = new File(contents, { readonly: true });
        } catch (err) {
          throw `couldn't preload '${key}: ${err}`;
        };
      };
    };
  };

  /** Prepare the filesystem for WASI shim context. */
  private async preopenFilesystem(load?: AdvancedExecutionOptions["rootfs"], stdin?: string | ArrayBuffer | Uint8Array): Promise<Fd[]> {
    // load files from OPFS
    let rootfs: RootFS = { };
    await this.preloadRootfs(rootfs, load);
    // encode stdin as buffer
    if (stdin === undefined || typeof stdin == "string") {
      stdin = new TextEncoder().encode(stdin);
    };
    // preopen file descriptors
    return <Fd[]>[
      new OpenFile(new File(stdin)),
      new OpenFile(new File([])), // stdout
      new OpenFile(new File([])), // stderr
      // new PreopenDirectory("/", rootfs), // TODO
    ];
  };

  /** Use the WASI shim's imports to provide a compatible syscall interface. */
  private systemInterface(shim: WASI, trace: boolean = false) { return {
    "wasi_unstable":          trace ? strace(shim.wasiImport, []) : shim.wasiImport,
    "wasi_snapshot_preview1": trace ? strace(shim.wasiImport, []) : shim.wasiImport,
  }};

  /** Run a WebAssembly binary with commandline arguments `args` and environment
   * variables `envs`. The `binary` is either a transferred `ArrayBuffer` or a filename
   * of a binary stored in OPFS. */
  public async run(
    id: string,
    wasm: WebAssembly.Module | BufferSource | string,
    args: string[],
    envs: string[],
    options?: AdvancedExecutionOptions,
    trace?: Trace & ProxyMarked,
    verbose: boolean = this.verbose,
  ): Promise<CompletedExecution> {
    try {
      trace?.now("worker: function top");

      // log the overall commandline to terminal and console
      if (true) { // TODO
        let cmdline = [args[0] || "<binary>", ...args.slice(1), ...envs];
        console.log(...this.prefix, { cmdline, options });
        // this.terminal.info(`${this.name}: ${id} ${JSON.stringify(cmdline)}`);
        trace?.now("worker: commandline logged");
      };

      // initialize filesystem for shim
      let files = await this.preopenFilesystem(options?.rootfs, options?.stdin);
      // if (verbose) console.debug(...this.prefix, "Preloaded files:", files);
      trace?.now("worker: filesystem prepared");

      // if `wasm` isn't a module yet, we need to compile it
      if (!(wasm instanceof WebAssembly.Module)) {
        if (typeof wasm === "string") wasm = await this.filesystem.getWasmModule(wasm);
        //* there's an open ticket for firefox where postMessage payloads over ~250 MB crash
        // https://bugzilla.mozilla.org/show_bug.cgi?id=1754400 (via https://blog.stackblitz.com/posts/supporting-firefox/)
        else wasm = await WebAssembly.compile(wasm);
        trace?.now("worker: wasm module compiled");
      };

      // prepare the shim
      let shim = new WASI(args, envs, files);
      let syscalls = this.systemInterface(shim, options?.strace);
      let instance: WebAssembly.Instance | null = null;
      trace?.now("worker: wasi shim prepared");

      // instantiate the webassembly module, with retries on OOM errors
      let retries = 10; let t0 = performance.now();
      for (let attempt = 0; attempt <= retries; attempt++) {
        try {
          instance = await WebAssembly.instantiate(wasm, syscalls);
          break; // if the above succeeded, we can exit the loop
        } catch (error) {
          instance = null;
          if (String(error).includes("Out of memory: Cannot allocate Wasm memory for new instance")) {
            trace?.now("worker: OOM, retry");
            let warning = (`OOM on ${attempt+1}. instantiation attempt after ${performance.now() - t0} ms`);
            console.warn(...this.prefix, warning);
            this.terminal.warn(this.name + ": " + warning);
            if (attempt === retries) throw error;
          } else {
            // wasn't OOM, immediately rethrow
            throw error;
          };
        };
        // wait 10, 20, 40, 80, 160, 320, ... ms
        await new Promise(r => setTimeout(r, 2**attempt));
      };
      if (instance === null) throw "WebAssembly module was not instantiated!";
      trace?.now("worker: module instantiated");

      //! FinalizationRegistry test to get notified when WebAssembly.Memory is collected
      // TODO: actually, the callbacks can also be used to "wake up" retrying workers!
      this.registry.register(instance.exports.memory, "WebAssembly.Memory has been collected!");

      // start the instance's main() and wait for it to exit
      let exitcode = 0;
      try {
        shim.start(instance as WasiInstance);
      } catch(error) {
        if (String(error).startsWith("exit with exit code")) {
          // parse the exitcode from exit() calls; those shouldn't throw
          exitcode = Number(String(error).substring(26));
        } else {
          // rethrow everything else
          throw error;
        };
      } finally {
        //! always explicitly null' the instance as a hint to garbage collector
        // this won't completely prevent out-of-memory errors but might make the GC run earlier
        instance = null;
      };
      trace?.now("worker: task completed");
      
      // format the output
      let output: CompletedExecution = {
        exitcode,
        // TODO: decoding is fine for now but generally shouldn't as outputs can be binary
        stdout: new TextDecoder().decode((<OpenFile>shim.fds[1]).file.data),
        stderr: new TextDecoder().decode((<OpenFile>shim.fds[2]).file.data),
        trace: await trace?.export(),
      };
      if (verbose) console.debug(...this.prefix, "Finished execution:", {
        stdout: output.stdout, stderr: output.stderr,
        // filesystem: (shim.fds[3] as PreopenDirectory).dir.contents,
      });
      if (options?.datafile) { // maybe copy an output file
        // TODO, BUG: if the binary exited non-sucessfully, the file may not exist here!
        // let f = options.datafile;
        // let dir = (shim.fds[3] as PreopenDirectory).dir.contents;
        // if (dir[f] === undefined) throw `requested datafile "${f}" not found!`;
        // output.datafile = (dir[f] as File).data;
      };
      if (verbose && trace) console.info(`Trace of ${id}:`, trace.export());
      if (verbose) console.info(...this.prefix, "Task output:", output);

      return output;

    } catch (err) {
      this.terminal.error(`${this.name}: ERROR: ${err}`);
      throw err;
    };
  };

};

expose(WASMRunner);
postMessage(workerReady);


/** Usage example:

// run the webassembly binary in worker
async function run(prefetch: boolean = true) {
  try {

    // abort if there are no runners in the pool yet
    if (pool.count < 1) {
      throw "there are no runners in the pool!"
    }

    const output = await pool.exec(async worker => {

      // log which method we're about to use
      terminal.log("Prepare WASMRunner request using " + (prefetch ? "prefetched ArrayBuffer." : "filename in OPFS."), LogType.Link);
  
      // maybe prefetch the binary and send to runner
      const binary = prefetch ? await (await fetch(BINARY)).arrayBuffer() : OPFS_FILE;
      const promise = worker.run(
        [ "wasm.exe", "print_envs_", "file:/shared.txt" ],
        [ "PROJECT=wasimoff", "SECRETKEY=hunter12" ],
        (typeof binary === "string") ? binary : transfer(binary, [binary]),
      );
  
      // the binary was fully transferred and is now empty in this context
      if (prefetch && (<ArrayBuffer>binary).byteLength != 0) throw new Error("oops. buffer should have been transferred!");
      terminal.log(`WASMRunner request sent to <${await worker.name}>.`, LogType.Info);
      terminal.log("WASM request sent");

      return promise;
    });

    // print stdout to terminal
    console.warn(output);
    terminal.log("WASMRunner output:\n" + output.split("\n").map(line => `> ${line}`).join("\n"), LogType.Success);

  } catch (err) {
    terminal.error(`WASMRunner: ${err}`);
  }
};

*/
