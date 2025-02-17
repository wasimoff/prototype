/// <reference lib="webworker" />
declare var self: DedicatedWorkerGlobalScope;
export {};

// TODO: use FinalizationRegistry to notify OOM'ed workers (https://github.com/wasimoff/prototype/blob/cf3b222aba5dd218040fcc6b15af425f0f95b35a/webprovider/src/worker/wasmrunner.ts#L52-L54)

import { WASI, File, OpenFile, PreopenDirectory, Fd, strace } from "@bjorn3/browser_wasi_shim";
import { ZipReader, Uint8ArrayReader, Uint8ArrayWriter, ZipWriter } from "@zip.js/zip.js";
import { expose, workerReady } from "./comlink.ts";
import { Inode } from "@bjorn3/browser_wasi_shim";
import { Directory } from "@bjorn3/browser_wasi_shim";
import { loadPyodide } from "pyodide";


/** Web Worker which runs WebAssembly modules with a WASI shim in a quasi threadpool. */
export class WasiWorker {

  constructor(
    private readonly index: number,
    private readonly verbose: boolean = false,
  ) { };

  // colorful console logging prefix
  private get logprefix() { return [ `%c[Worker ${this.index}]`, "color: #f03a5f;" ]; }


  /** Run a WebAssembly module with a WASI shim with commandline arguments, environment
   * variables etc. The binary can be either a precompiled module or raw bytes. */
  public async runWasip1(id: string, task: Wasip1TaskParams): Promise<Wasip1TaskResult> {
    try {

      // log the overall commandline to dev console
      if (this.verbose) {
        let cmdline = [...task.envs, task.argv[0] || "<binary>", ...task.argv.slice(1)];
        console.info(...this.logprefix, id, cmdline);
      };

      // initialize filesystem for shim
      let fds = await preopenFilesystem(task);

      // if `wasm` isn't a module yet, we need to compile it
      if (!(task.wasm instanceof WebAssembly.Module)) {
        //! there's an open ticket for firefox where postMessage payloads over ~250 MB crash, so be careful
        // https://bugzilla.mozilla.org/show_bug.cgi?id=1754400 (via https://blog.stackblitz.com/posts/supporting-firefox/)
        task.wasm = await WebAssembly.compile(task.wasm);
      };

      // prepare the browser_wasi_shim
      let shim = new WASI(task.argv, task.envs, fds, { debug: false });
      patchWasiPollOneoff(shim); // fixes some async IO, like time.Sleep() in Go
      let syscalls = {
        "wasi_snapshot_preview1": task.strace ? strace(shim.wasiImport, []) : shim.wasiImport,
        "wasi_unstable":          task.strace ? strace(shim.wasiImport, []) : shim.wasiImport,
      };
      
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
            let elapsed = performance.now() - t0;
            console.warn(...this.logprefix, `WebAssembly.instantiate OOM, attempt ${attempt}, after ${elapsed} ms`);
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

      // start the instance's main() and wait for it to exit
      let returncode = 0;
      try {
        type Wasip1Instance = { exports: { memory: WebAssembly.Memory, _start: () => unknown } };
        returncode = shim.start(instance as Wasip1Instance);
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

      // format the result
      let result: Wasip1TaskResult = {
        returncode,
        stdout: (<OpenFile>shim.fds[1]).file.data,
        stderr: (<OpenFile>shim.fds[2]).file.data,
      };
      if (task.artifacts !== undefined && task.artifacts.length > 0) {
        result.artifacts = await compressArtifacts(shim.fds[3] as PreopenDirectory, task.artifacts);
      };
      return result;
    } catch (err) {
      console.error(...this.logprefix, "oops:", err);
      throw err;
    };
  };


  /** Run a Python script through Pyodide. Give the plaintext script and any known imported
   * packages in the parameters. If the last statement returns a result, it is pickled back. */
  public async runPyodide(id: string, task: PyodideTaskParams): Promise<PyodideTaskResult> {
    try {

      // load a fresh pyodide instance to avoid polluting context between tasks
      console.log(...this.logprefix, "loading Pyodide for", id);
      let t0 = performance.now();
      const py = await loadPyodide({
        jsglobals: new Map(), // do not pollute worker context
        fullStdLib: false, // probably a little faster
        checkAPIVersion: true, // must be this exact version
        packages: task.packages, // preload some packages explicitly
      });
      console.debug(...this.logprefix, "loading took", performance.now() - t0, "ms");

      // setup the io buffers
      let stdout = new Uint8Array();
      let stderr = new Uint8Array();
      py.setStdout({ write: (more: Uint8Array) => {
        let larger = new Uint8Array(stdout.length + more.length);
        larger.set(stdout, 0); larger.set(more, stdout.length);
        stdout = larger;
        return more.length;
      }});
      py.setStderr({ write: (more: Uint8Array) => {
        let larger = new Uint8Array(stderr.length + more.length);
        larger.set(stderr, 0); larger.set(more, stderr.length);
        stderr = larger;
        return more.length;
      }});

      // run the script
      await py.loadPackagesFromImports(task.script);
      let ret = py.runPython(task.script);
      let result: PyodideTaskResult = {
        stdout, stderr, version: py.version,
      };

      // maybe pickle the last line result
      if (ret !== undefined) {
        console.debug(...this.logprefix, "Pickling the result ...");
        await py.loadPackage("cloudpickle");
        let r = py.runPython("import cloudpickle as cp; cp.dumps(ret)", { locals: new Map([["ret", ret]]) as any });
        if (r.type !== "bytes") throw "couldn't pickle the execution result";
        result.pickle = r.toJs();
        try {
          // try to clean up
          r.destroy();
          ret.destroy();
        } catch { };
      }
      return result;

    } catch (err) {
      console.error(...this.logprefix, "oops:", err);
      throw err;
    };
  };


}; // WasiWorker

// only expose if we're actually started in a worker and not just being imported
if (self.constructor.name === "DedicatedWorkerGlobalScope" && self.postMessage !== undefined) {
  expose(WasiWorker, self);
  postMessage(workerReady);
};


// ------------------------- typings ------------------------- //

/** Parameters for a wasip1 task. */
export type Wasip1TaskParams = {
  /** The WebAssembly executable itself, either precompiled module or a binary source. */
  wasm: WebAssembly.Module | BufferSource;
  /** Commandline arguments. */
  argv: string[];
  /** Environment variables in a `KEY=value` mapping. */
  envs: string[];
  /** Put something on `stdin`, instead of an empty file. */
  stdin?: Uint8Array;
  /** Load files for preloaded filesystem from a zip archive. */
  rootfs?: Uint8Array;
  /** Send back a zip archive with artifacts after successful execution. */
  artifacts?: string[];
  /** Wrap the WASI imports in `strace` for improved debug visibility. */
  strace?: boolean;
};

/** Result of a wasip1 task. */
export type Wasip1TaskResult = {
  /** The returned exit code, where `0` usually indicates success. */
  returncode: number,
  /** Standard output as bytes. */
  stdout: Uint8Array,
  /** Standard error as bytes. */
  stderr: Uint8Array,
  /** Packed artifacts that were requested. */
  artifacts?: Uint8Array;
};

/** Parameters for a Pyodide task. */
export type PyodideTaskParams = {
  /** The Python script to execute. Last statement may be pickled back. */
  script: string;
  /** Preload known packages more efficiently during instantiation. */
  packages: string[];
};

/** Result of a Pyodide task. */
export type PyodideTaskResult = {
  /** Standard output as bytes. */
  stdout: Uint8Array;
  /** Standard error as bytes. */
  stderr: Uint8Array;
  /** Pickled result from last statement in script, if any. */
  pickle?: Uint8Array;
  /** Pyodide version, might be important to unpickle. */
  version: string;
};



//
// -------------------- filesystem utils --------------------

/** Prepare the filesystem for WASI shim. */
async function preopenFilesystem(task: Wasip1TaskParams): Promise<Fd[]> {
  // prepare a rootfs and optionally extract zip file
  let rootfs = new PreopenDirectory(".", new Map());
  if (task.rootfs !== undefined)
    rootfs = await extractRootfs(task.rootfs);
  // return file descriptors
  return [
    new OpenFile(new File(task.stdin || [])),
    new OpenFile(new File([])), // stdout
    new OpenFile(new File([])), // stderr
    rootfs,
  ];
};

/** Extract a ZipReader to a preopened directory for the browser_wasi_shim */
async function extractRootfs(archive: Uint8Array): Promise<PreopenDirectory> {
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

    }; // path.entries
  }; // zip.getEntriesGenerator

  // return nested map as preopened directory
  return new PreopenDirectory("/", root);
};

/** Pack requested artifacts with a ZipWriter to send back. */
async function compressArtifacts(dir: PreopenDirectory, artifacts: string[]): Promise<Uint8Array> {
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
};


//
// -------------------- patches --------------------

import { wasi } from "@bjorn3/browser_wasi_shim";

// Workaround for https://github.com/bjorn3/browser_wasi_shim/issues/14
// from: https://gist.github.com/igrep/0cf42131477422ebba45107031cd964c
export function patchWasiPollOneoff(self: WASI): void {
  self.wasiImport.poll_oneoff = ((
    inPtr: number,
    outPtr: number,
    nsubscriptions: number,
    sizeOutPtr: number,
  ): number => {
    if (nsubscriptions < 0) {
      return wasi.ERRNO_INVAL;
    }

    const size_subscription = 48;
    const subscriptions = new DataView(
      self.inst.exports.memory.buffer,
      inPtr,
      nsubscriptions * size_subscription,
    );

    const size_event = 32;
    const events = new DataView(
      self.inst.exports.memory.buffer,
      outPtr,
      nsubscriptions * size_event,
    );

    for (let i = 0; i < nsubscriptions; ++i) {
      const subscription_userdata_offset = 0;
      const userdata = subscriptions.getBigUint64(
        i * size_subscription + subscription_userdata_offset,
        true,
      );

      const subscription_u_offset = 8;
      const subscription_u_tag = subscriptions.getUint8(
        i * size_subscription + subscription_u_offset,
      );
      const subscription_u_tag_size = 1;

      const event_userdata_offset = 0;
      const event_error_offset = 8;
      const event_type_offset = 10;
      const event_fd_readwrite_nbytes_offset = 16;
      const event_fd_readwrite_flags_offset = 16 + 8;

      events.setBigUint64(
        i * size_event + event_userdata_offset,
        userdata,
        true,
      );
      events.setUint32(
        i * size_event + event_error_offset,
        wasi.ERRNO_SUCCESS,
        true,
      );

      function assertOpenFileAvailable(): OpenFile {
        const fd = subscriptions.getUint32(
          i * size_subscription +
            subscription_u_offset +
            subscription_u_tag_size,
          true,
        );
        const openFile = self.fds[fd];
        if (!(openFile instanceof OpenFile)) {
          throw new Error(`FD#${fd} cannot be polled!`);
        }
        return openFile;
      }
      function setEventFdReadWrite(size: bigint): void {
        events.setUint16(
          i * size_event + event_type_offset,
          wasi.EVENTTYPE_FD_READ,
          true,
        );
        events.setBigUint64(
          i * size_event + event_fd_readwrite_nbytes_offset,
          size,
          true,
        );
        events.setUint16(
          i * size_event + event_fd_readwrite_flags_offset,
          0,
          true,
        );
      }
      switch (subscription_u_tag) {
        case wasi.EVENTTYPE_CLOCK:
          events.setUint16(
            i * size_event + event_type_offset,
            wasi.EVENTTYPE_CLOCK,
            true,
          );
          break;
        case wasi.EVENTTYPE_FD_READ:
          const fileR = assertOpenFileAvailable();
          setEventFdReadWrite(fileR.file.size);
          break;
        case wasi.EVENTTYPE_FD_WRITE:
          // I'm not sure why, but an unavailable (already closed) FD is referenced here. So don't call assertOpenFileAvailable.
          setEventFdReadWrite(1n << 31n);
          break;
        default:
          throw new Error(`Unknown event type: ${subscription_u_tag}`);
      }
    }

    const size_size = 4;
    const outNSize = new DataView(
      self.inst.exports.memory.buffer,
      sizeOutPtr,
      size_size,
    );
    outNSize.setUint32(0, nsubscriptions, true);
    return wasi.ERRNO_SUCCESS;
  }) as (...args: unknown[]) => unknown;
}
