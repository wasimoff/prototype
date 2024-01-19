import { computed, ref, shallowRef } from "vue";
import { defineStore } from "pinia";
import { LRUCache } from "lru-cache";

// get a top-level handle for the origin-private filesystem
const opfs = await navigator.storage.getDirectory();

// manage origin-private filesystem (OPFS) access in a single store
export const useFilesystem = defineStore("Filesystem", () => {

  // TODO: use subdirectories for binaries and data files
  // TODO: store with content-addressing, e.g. by SHA256 hash

  // colorful prefix for logging
  const prefix = [ "%c Filesystem ", "background: purple; color: white;" ];

  // enable verbose logging
  let verbose = ref(false);

  // use a least-recently-used (LRU) cache for the retrieved binaries
  let cache = new LRUCache<string, ArrayBuffer>({
    max: 25, // at most 25 items
    ttl: 60 * 1000, // consider stale after a minute
    fetchMethod: async filename => await getBuffer(filename), // fetch files from OPFS
  });

  // a separate cache for compiled WebAssembly modules
  //! you should take care not to use `cache` and `wasmcache` with the same binaries
  let wasmcache = new LRUCache<string, WebAssembly.Module>({
    max: 25, // at most 25 modules
    ttl: 60 * 1000, // consider stale after a minute
    // fetch and compile modules from OPFS
    fetchMethod: async (filename) => {
      let file = await getFile(filename);
      if (file.type !== "application/wasm") throw new Error("this file isn't a WebAssembly module");
      return await WebAssembly.compileStreaming(asResponse(file));
    },
  });

  /** Fetch an arbitrary file from URL and write to a file in OPFS. */
  async function download(url: string, filename: string, type?: string) {
    // fire off the request
    const request = window.fetch(url);
    // open writable stream of file to download to
    const file = await opfs.getFileHandle(filename, { create: true });
    const stream = await file.createWritable();
    // check if request is OK and content-type is as expected
    const response = await request;
    if (!response.ok)
      throw new Error(`request failed: ${response.status} ${response.statusText}`);
    if (type && response.headers.get("content-type")?.toLowerCase() !== "application/wasm")
      throw new Error(`fetched object has unexpected type: ${response.headers.get("content-type")}`);
    // pipe the response body to file stream
    await response.body!.pipeTo(stream);
    if (verbose.value) console.log(...prefix, "Downloaded", new URL(url, import.meta.url), `to /${filename}.`);
    // reply with the downloaded size
    return (await file.getFile()).size;
  };

  /** Store a given byte array into the filesystem. */
  async function store(blob: BufferSource, filename: string) {
    const handle = await opfs.getFileHandle(filename, { create: true });
    const file = await handle.createWritable({ keepExistingData: false });
    await file.write(blob);
    await file.close();
    let size = (await handle.getFile()).size;
    if (verbose.value) console.log(...prefix, `Stored ${size} bytes in /${filename}.`);
  };

  /** Retrieve a file from OPFS. */
  async function getFile(filename: string) {
    const handle = await opfs.getFileHandle(filename);
    return handle.getFile();
  };

  /** Retrieve a file's contents directly from OPFS. */
  async function getBuffer(filename: string) {
    return (await getFile(filename)).arrayBuffer();
  };

  /** Return a file's readable stream within a faked fetch Response with
   * the appropriate content-type. This allows streaming compilation. */
  async function asResponse(file: File) {
    return new Response(file.stream(), {
      status: 200,
      headers: { "content-type": file.type },
    });
  };

  /** Retrieve a cached or fetch a file's contents from OPFS. */
  async function getCached(filename: string) {
    // if the file does not exist (would have been `undefined`) it throws instead
    return (await cache.fetch(filename))!;
  };

  /** Retrieve a WebAssembly binary as compiled module (may be cached). */
  async function getWasmModule(filename: string) {
    return (await wasmcache.fetch(filename))!;
  };

  /** List files in OPFS root. */
  async function ls() {
    const entries: File[] = [];
    for await (const name of (<any>opfs).keys() as string) {
      let handle = await opfs.getFileHandle(name);
      let file = await handle.getFile();
      entries.push(file);
    };
    return entries;
  };

  /** Remove a particular file in OPFS root. */
  async function rm(filename: string) {
    await opfs.removeEntry(filename);
    cache.delete(filename);
    if (verbose.value) console.log(...prefix, `Deleted /${filename}.`);
  };

  /** Remove all files in OPFS root. */
  async function rmrf() {
    let files = (await ls()).map(file => file.name);
    for (let file of files) await rm(file);
    return files;
  };

  return { download, store, getFile, getBuffer, asResponse, getCached, getWasmModule, ls, rm, rmrf };
});