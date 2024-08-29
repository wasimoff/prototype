import { ProviderStorage } from "./index.ts";
import { LRUCache } from "lru-cache";

const logprefix = [ "%c OPFS Storage ", "background: purple; color: white;" ];

export class OpfsStorage implements ProviderStorage {

  // we need async initialization, therefore disallow direct constructor usage
  private constructor(
    private readonly handle: FileSystemDirectoryHandle,
    public readonly path: string,
  ) { }


  /** Initialize a new Filesystem with OPFS backing. */
  static async open(directory?: string | FileSystemDirectoryHandle) {
    let opfs = await navigator.storage.getDirectory();
    let storage: OpfsStorage;
    // easy mode: open the root
    if (directory === undefined || directory === "/") {
      storage = new OpfsStorage(opfs, "/");
    } else {
      // directory is a path, open each fragment until we reach its handle
      if (typeof directory === "string") {
        let handle = opfs; // start at root
        for (let fragment of directory.split("/").filter(f => f != "")) {
          handle = await handle.getDirectoryHandle(fragment, { create: true });
        }
        directory = handle;
      }
      // (now) directory is a handle, resolve its path and open
      let path = await opfs.resolve(directory);
      if (path === null) throw "given DirectoryHandle is not in OPFS";
      storage = new OpfsStorage(directory, `/${path.join("/")}/`)
    }
    console.log(...logprefix, `opened Origin-Private Filesystem at "${storage.path}"`);
    return storage;
  }


  // ------------------ get data ------------------ //

  /** Get items in directory. */
  async ls() {
    let items: (FileSystemDirectoryHandle | File)[] = [ ];
    for await (let it of (this.handle as any).values()) {
      if (it instanceof FileSystemFileHandle) it = await it.getFile();
      items.push(it);
    };
    return items;
  };

  /** Get files in directory. */
  async lsf() {
    return (await this.ls()).filter(e => e instanceof File) as File[];
  }

  /** Retrieve a file. */
  async getFile(filename: string) {
    let handle = await this.handle.getFileHandle(filename);
    return await handle.getFile();
  };

  /** Retrieve a file's contents directly. */
  async getBuffer(filename: string) {
    let file = await this.getFile(filename);
    return await file.arrayBuffer();
  };

  /** Retrieve a WebAssembly binary as compiled module (will be cached). */
  async getWasmModule(filename: string) {
    return (await this.wasmcache.fetch(filename))!;
  };

  /** The cache for compiled WebAssembly modules. */
  //! you should take care not to use a "file" cache and `wasmcache` with the same binaries
  private wasmcache = new LRUCache<string, WebAssembly.Module>({
    max: 25, // at most 25 modules
    ttl: 10 * 60 * 1000, // consider stale after 10 minutes
    // fetch and compile modules from OPFS
    fetchMethod: async (filename) => await this.compileStreaming(filename),
  });

  /** Compile a `WebAssembly.Module` by opening a file in a streaming fashion. */
  private async compileStreaming(filename: string) {
    console.log(...logprefix, `compile WebAssembly module ${this.path}${filename}`);
    // fetch the file from opfs and check if it's wasm
    let file = await (await this.handle.getFileHandle(filename)).getFile();
    if (file.type !== "application/wasm") throw new Error("this file isn't a WebAssembly module");
    // fake a fetch response to facilitate streaming
    let stream = new Response(file.stream(), { status: 200, headers: { "content-type": file.type } });
    // start the streaming compilation
    return await WebAssembly.compileStreaming(stream);
  };


  // ------------------ modify data ------------------ //

  /** Remove a particular file. */
  async rm(filename: string) {
    console.log(...logprefix, `delete ${this.path}${filename}`);
    await this.handle.removeEntry(filename);
    return this.wasmcache.delete(filename);
  };

  /** Remove all files in directory. */
  async prune() {
    let files = (await this.lsf()).map(f => f.name);
    for (let file of files) await this.rm(file);
    return files;
  };

  /** Store a given BufferSource into the filesystem. */
  async store(blob: BufferSource, filename: string) {
    console.log(...logprefix, `store ${blob.byteLength} bytes in ${this.path}${filename}`);
    let handle = await this.handle.getFileHandle(filename, { create: true });
    let file = await handle.createWritable({ keepExistingData: false });
    await file.write(blob);
    await file.close();
    return await handle.getFile();
  };

  /** Fetch an arbitrary file from URL and write to a file in directory. */
  async download(url: string, filename: string) {
    console.log(...logprefix, `download ${new URL(url, import.meta.url)} to ${this.path}${filename}`);
    // start the request in background
    let request = window.fetch(url);
    // open writable stream of file to download to
    let file = await this.handle.getFileHandle(filename, { create: true });
    let stream = await file.createWritable();
    let response = await request;
    // check if request is OK and content-type is as expected
    if (!response.ok) throw new Error(`request failed: ${response.status} ${response.statusText}`);
    let type = response.headers.get("content-type")?.toLowerCase();
    if (type !== "application/wasm") throw new Error(`fetched object has unexpected type: ${type}`);
    // pipe the response body to file stream and return the file
    await response.body!.pipeTo(stream);
    await stream.close();
    return await file.getFile();
  };

}