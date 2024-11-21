import { ProviderStorageFileSystem } from "./index.ts";

const logprefix = [ "%c[OriginPrivateFileSystem]", "color: purple;" ];

export class OriginPrivateFileSystem implements ProviderStorageFileSystem {

  // we need async initialization, therefore disallow direct constructor usage
  private constructor(
    private readonly handle: FileSystemDirectoryHandle,
    public readonly path: string,
  ) { }

  /** Initialize a new Filesystem with OPFS backing. */
  static async open(directory?: string | FileSystemDirectoryHandle) {
    let opfs = await navigator.storage.getDirectory();
    let storage: OriginPrivateFileSystem;
    // easy mode: open the root
    if (directory === undefined || directory === "/") {
      storage = new OriginPrivateFileSystem(opfs, "/");
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
      storage = new OriginPrivateFileSystem(directory, `/${path.join("/")}/`)
    }
    console.log(...logprefix, `opened OPFS at "${storage.path}"`);
    return storage;
  }


  // list all files in directory
  async list(): Promise<string[]> {
    let files: string[] = [];
    for await (let it of (this.handle as any).values()) {
      // can also be FileSystemDirectoryHandle, but we ignore those here
      if (it instanceof FileSystemFileHandle) files.push(it.name);
    };
    return files;
  };

  // return a specific file
  async get(filename: string): Promise<File | undefined> {
    try {
      return this.handle.getFileHandle(filename).then(h => h.getFile());
    } catch (err) {
      return undefined;
    };
  };

  // store a file in filesystem
  async put(filename: string, file: File): Promise<File> {
    console.debug(...logprefix, `store:`, file);
    let handle = await this.handle.getFileHandle(filename, { create: true });
    let writer = await handle.createWritable({ keepExistingData: false });
    await writer.write(await file.arrayBuffer());
    await writer.close();
    return await handle.getFile();
  };

  // remove a particular file
  async rm(filename: string): Promise<boolean> {
    console.debug(...logprefix, `delete:`, filename);
    let had = (await this.list()).includes(filename);
    await this.handle.removeEntry(filename);
    return had;
  };



  // ------------------- old functions ------------------- //

  /** Compile a `WebAssembly.Module` by opening a file in a streaming fashion. */
  async compileStreaming(filename: string) {
    console.log(...logprefix, `compile WebAssembly module ${this.path}${filename}`);
    // fetch the file from opfs and check if it's wasm
    let file = await (await this.handle.getFileHandle(filename)).getFile();
    if (file.type !== "application/wasm") throw new Error("this file isn't a WebAssembly module");
    // fake a fetch response to facilitate streaming
    let stream = new Response(file.stream(), { status: 200, headers: { "content-type": file.type } });
    // start the streaming compilation
    return await WebAssembly.compileStreaming(stream);
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
