/// <reference lib="webworker" />
declare var self: DedicatedWorkerGlobalScope;
export {};

// TODO: in Deno, replace browser_wasi_shim with std module https://deno.land/std@0.93.0/wasi/README.md?
// TODO: accept rootfs and return artifacts as zip using zip.js https://gildas-lormeau.github.io/zip.js/api/
// TODO: use FinalizationRegistry to notify OOM'ed workers (https://github.com/wasimoff/prototype/blob/cf3b222aba5dd218040fcc6b15af425f0f95b35a/webprovider/src/worker/wasmrunner.ts#L52-L54)

import { WASI, File, OpenFile, PreopenDirectory, Fd, strace } from "@bjorn3/browser_wasi_shim";
import { ZipReader, Uint8ArrayReader, Uint8ArrayWriter, ZipWriter } from "@zip.js/zip.js";
import { expose, workerReady } from "./comlink.ts";
import { Inode } from "@bjorn3/browser_wasi_shim";
import { Directory } from "@bjorn3/browser_wasi_shim";

// be more verbose with the messages
const VERBOSE = true;

/** Web Worker which runs WebAssembly modules with a WASI shim in a quasi threadpool. */
export class WasiWorker {

  constructor(
    private readonly index: number,
  ) { };

  // colorful console logging prefix
  private get logprefix() { return [ `%c WasiWorker ${this.index} `, "background: #f03a5f; color: white;" ]; }

  // TODO: shim the trace function to not rip out all the statements completely
  private trace(msg: string) { if (false) console.debug(...this.logprefix, msg); };



  /** Run a WebAssembly module with a WASI shim with commandline arguments, environment
   * variables etc. The binary can be either a precompiled module or raw bytes. */
  public async run(id: string, task: WasiTaskExecution): Promise<WasiTaskResult> {
    try {
      this.trace("worker: function top");

      // log the overall commandline to terminal and console
      if (VERBOSE) { // TODO
        let cmdline = [...task.envs, task.argv[0] || "<binary>", ...task.argv.slice(1)];
        console.debug(...this.logprefix, id, cmdline);
        this.emit("cmdline", { id, cmdline });
        this.trace("worker: commandline logged");
      };

      // initialize filesystem for shim
      let fds = await this.preopenFilesystem(task);
      this.trace("worker: filesystem prepared");

      // if `wasm` isn't a module yet, we need to compile it
      if (!(task.wasm instanceof WebAssembly.Module)) {
        //! there's an open ticket for firefox where postMessage payloads over ~250 MB crash, so be careful
        // https://bugzilla.mozilla.org/show_bug.cgi?id=1754400 (via https://blog.stackblitz.com/posts/supporting-firefox/)
        task.wasm = await WebAssembly.compile(task.wasm);
        this.trace("worker: wasm module compiled");
      };
      this.trace("worker: wasm module prepared");

      // prepare the browser_wasi_shim
      let shim = new WASI(task.argv, task.envs, fds, { debug: false });
      let syscalls = {
        "wasi_snapshot_preview1": task.strace ? strace(shim.wasiImport, []) : shim.wasiImport,
        "wasi_unstable":          task.strace ? strace(shim.wasiImport, []) : shim.wasiImport,
      };
      this.trace("worker: wasi shim prepared");
      
      // instantiate the webassembly module, with retries on OOM errors
      let instance: WebAssembly.Instance | null = null;
      let retries = 10; let t0 = performance.now();
      for (let attempt = 0; attempt <= retries; attempt++) {
        try {
          instance = await WebAssembly.instantiate(task.wasm, syscalls);
          break; //? if the above succeeded, we can exit the loop
        } catch (error) {
          instance = null;
          if (String(error).includes("Out of memory: Cannot allocate Wasm memory for new instance")) {
            this.trace("worker: OOM, retry");
            let elapsed = performance.now() - t0;
            console.warn(...this.logprefix, `OOM, attempt ${attempt}, at ${elapsed} ms`);
            this.emit("oom", { id, attempt, elapsed });
            if (attempt === retries) throw error;
          } else {
            // this wasn't OOM, immediately rethrow
            throw error;
          };
        };
        // wait 10, 20, 40, 80, 160, 320, ... ms
        await new Promise(r => setTimeout(r, 2**attempt));
      };
      if (instance === null) throw "WebAssembly module was not instantiated!";
      this.trace("worker: module instantiated");

      // start the instance's main() and wait for it to exit
      let returncode = 0;
      try {
        shim.start(instance as WasiInstance);
      } catch(error) {
        if (String(error).startsWith("exit with exit code")) {
          // parse the exitcode from exit() calls; those shouldn't throw
          returncode = Number(String(error).substring(26));
        } else {
          // rethrow everything else
          throw error;
        };
      } finally {
        //! always explicitly null' the instance as a hint to garbage collector
        // this won't completely prevent out-of-memory errors but might make the GC run earlier
        instance = null;
      };
      this.trace("worker: task completed");
      
      // format the output
      let output: WasiTaskResult = {
        returncode,
        // TODO: decoding is fine for now but generally shouldn't as outputs can be binary
        stdout: (<OpenFile>shim.fds[1]).file.data,
        stderr: (<OpenFile>shim.fds[2]).file.data,
        // TODO: re-add trace
      };
      if (VERBOSE) console.debug(...this.logprefix, "exit code", output.returncode);
      // {
      //   returncode,
      //   stdout: output.stdout, stderr: output.stderr,
      //   // filesystem: (shim.fds[3] as PreopenDirectory).dir.contents,
      // });
      // if (options?.datafile) { // maybe copy an output file
        // TODO, BUG: if the binary exited non-sucessfully, the file may not exist here!
        // let f = options.datafile;
        // let dir = (shim.fds[3] as PreopenDirectory).dir.contents;
        // if (dir[f] === undefined) throw `requested datafile "${f}" not found!`;
        // output.datafile = (dir[f] as File).data;
      // };
      // if (verbose && trace) console.info(`Trace of ${id}:`, trace.export());
      // if (verbose) console.info(...this.logprefix, "Task output:", output);

      if (task.artifacts !== undefined) {
        output.artifacts = await this.compressArtifacts(shim.fds[3] as PreopenDirectory, task.artifacts);
      };

      return output;

    } catch (err) {
      this.emit("failure", { id, err: String(err) });
      throw err;
    };
  };


  /** Prepare the filesystem for WASI shim. */
  private async preopenFilesystem(task: WasiTaskExecution): Promise<Fd[]> {
    // extract rootfs or use an empty one
    let rootfs: PreopenDirectory;
    if (task.rootfs !== undefined) {
      rootfs = await this.extractRootfs(task.rootfs);
    } else {
      rootfs = new PreopenDirectory(".", new Map());
    };
    // return file descriptors
    return [
      new OpenFile(new File(task.stdin || [])),
      new OpenFile(new File([])), // stdout
      new OpenFile(new File([])), // stderr
      rootfs,
    ];
  };

  /** Extract a ZipReader to a preopened directory for the browser_wasi_shim */
  private async extractRootfs(archive: Uint8Array): Promise<PreopenDirectory> {
    const zip = new ZipReader(new Uint8ArrayReader(archive));

    // TODO: can we use create_entry_for_path directly?
    let root = new Map<string, Inode>();
    for await (const entry of zip.getEntriesGenerator()) {
      let pwd = root; // current working dir

      // descend to the corrent node, creating directories along the way
      if (entry.filename.endsWith("/")) entry.filename = entry.filename.slice(0, -1);
      const path = entry.filename.split("/");
      for (const [i, name] of path.entries()) {

        // last path component => set the contents
        if (i === path.length - 1) {
          if (entry.directory) {
            // set an empty directory
            pwd.set(name, new Directory(new Map()));
          } else {
            // get contents and insert a File
            let bufwriter = new Uint8ArrayWriter();
            await entry.getData!(bufwriter);
            pwd.set(name, new File(await bufwriter.getData()));
          };
          continue;
        } else {
          // create if directory does not exist
          if (!(pwd.get(name) instanceof Directory)) {
            pwd.set(name, new Directory(new Map()));
          };
          // descend into directory
          pwd = (pwd.get(name) as Directory).contents;
        };

      };

    };

    // return nested map as preopened directory
    return new PreopenDirectory("/", root);

  };

  /** Pack requested artifacts with a ZipWriter to send back. */
  private async compressArtifacts(dir: PreopenDirectory, artifacts: string[]): Promise<Uint8Array> {
    let zip = new ZipWriter(new Uint8ArrayWriter());

    // add all requested files
    await Promise.all(artifacts.map(filename => {
      // lookup the file in rootfs
      if (filename.startsWith("/")) filename = filename.slice(1);
      let { inode_obj: entry } = dir.path_lookup(filename, 0);
      if (entry instanceof File) {
        return zip.add(filename, new Uint8ArrayReader(entry.data), { useWebWorkers: false });
      } else {
        return zip.add(filename, undefined, { directory: true, useWebWorkers: false });
      };
    }));

    // finish the file and return its contents
    return await zip.close();
  }


  // private broadcast = new BroadcastChannel("WasiWorkerBroadcast");
  private emit<T extends keyof WasiWorkerMessages>(type: T, payload: WasiWorkerMessages[T]) {
  //   this.broadcast.postMessage({ name: this.name, type, payload } as SomeWasiWorkerMessage);
    self.postMessage({ name: String(this.index), type, payload } as SomeWasiWorkerMessage);
  };

};

// only expose if we're actually started in a worker and not just being imported
if (self.constructor.name === "DedicatedWorkerGlobalScope" && self.postMessage !== undefined) {
  expose(WasiWorker, self);
  postMessage(workerReady);
};


// ------------------------- typings ------------------------- //

// WebAssembly WASI instances have an exported `_start` method
export type WasiInstance = { exports: { memory: WebAssembly.Memory, _start: () => unknown } };

/** Arguments for a WASI task executions. */
// TODO: fully reuse the protobuf definitions?
export type WasiTaskExecution = {

  /** The WebAssembly executable itself, either precompiled module or a binary source. */
  wasm: WebAssembly.Module | BufferSource;
  /** Commandline arguments. */
  argv: string[];
  /** Environment variables in a `KEY=value` mapping. */
  envs: string[];

  /** Put something on `stdin`, instead of an empty file. */
  stdin?: Uint8Array;
  /** Load files for preloaded filesystem from an archive. */
  rootfs?: Uint8Array;
  /** Send back an archive with artifacts after successful execution. */
  artifacts?: string[];
  /** Wrap the WASI imports in `strace` for improved debug visibility. */
  strace?: boolean;

};

/** Result of a WASI task execution. */
export type WasiTaskResult = {

  /** The returned exit code, where `0` usually indicates success. */
  returncode: number,
  /** Standard output, decoded as a string. */
  stdout: Uint8Array,
  /** Standard error, decoded as a string. */
  stderr: Uint8Array,

  /** Packed artifacts that were requested. */
  artifacts?: Uint8Array;

  // TODO:
  // trace?: ExportedTrace,    // a trace of events with microsecond unix epochs

};


// ------------------------- messages ------------------------- //

export type WasiWorkerMessages = {

  /** Output the assembled commandline with argv and envs. */
  "cmdline": { id: string, cmdline: string[] };
  
  /** Oops, error during execution. */
  "failure": { id: string, err: string };
  
  /** Error: we had an out-of-memory error when trying to instantiate the module. */
  "oom": { id: string, attempt: number, elapsed: number };

};

export type SomeWasiWorkerMessage = {
  [K in keyof WasiWorkerMessages]: { type: K, name: string, payload: WasiWorkerMessages[K] }
}[keyof WasiWorkerMessages];
