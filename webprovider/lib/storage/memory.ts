import { ProviderStorage } from "./index.ts";
import { LRUCache } from "lru-cache";

const logprefix = [ "%c Memory Storage ", "background: purple; color: white;" ];

export class InMemoryStorage implements ProviderStorage {

  readonly path = ":memory:";

  // just keep file buffers in a map
  private storage = new Map<string, ArrayBuffer>();

  // cache compiled webassembly modules
  private wasmCache = new LRUCache<string, WebAssembly.Module>({
    max: 5, ttl: 2*60*1000, // five modules, stale after two minutes
    fetchMethod: async (filename) => await this.compile(filename),
  });

  // list files
  async lsf() {
    let files = <File[]>[];
    for (let [filename, buffer] of this.storage.entries()) {
      files.push(new File([buffer], filename));
    };
    return files;
  };

  // return files from map directly
  async getBuffer(filename: string) {
    return this.storage.get(filename);
  }

  // return a compiled and cached module
  async getWasmModule(filename: string) {
    return this.wasmCache.fetch(filename);
  };

  // compile a buffer to wasm module
  private async compile(filename: string) {
    let file = this.storage.get(filename);
    if (file === undefined) return undefined;
    return WebAssembly.compile(file);
  };

  // store a new file in the map
  async store(buf: ArrayBuffer, filename: string) {
    console.log(...logprefix, `store ${filename}, ${buf.byteLength} bytes`);
    this.storage.set(filename, buf);
    return new File([buf], filename);
  };

  // remove a file
  async rm(filename: string) {
    return this.storage.delete(filename);
  };

  // remove all files
  async prune() {
    let files = [...this.storage.keys()];
    this.storage.clear();
    return files;
  };

}
