

export interface WasimoffStorage {

  lsf(): Promise<File[]>;
  store(buf: ArrayBuffer, filename: string): Promise<File>;

  getBuffer(filename: string): Promise<ArrayBuffer | undefined>;
  getWasmModule(filename: string): Promise<WebAssembly.Module | undefined>;


}

import { LRUCache } from "lru-cache";

export class InMemoryStorage implements WasimoffStorage {

  // just keep file buffers in a map
  private storage = new Map<string, ArrayBuffer>();

  // cache compiled webassembly modules
  private wasmCache = new LRUCache<string, WebAssembly.Module>({
    max: 5, ttl: 2*60*1000, // five modules, stale after two minutes
    fetchMethod: async (filename) => await this.compile(filename),
  });

  async lsf() {
    let files = <File[]>[];
    for (let [filename, buffer] of this.storage.entries()) {
      files.push(new File([buffer], filename));
    };
    return files;
  };

  async getBuffer(filename: string) {
    return this.storage.get(filename);
  }

  async getWasmModule(filename: string) {
    return this.wasmCache.fetch(filename);
  };

  async compile(filename: string) {
    let file = this.storage.get(filename);
    if (file === undefined) return undefined;
    return WebAssembly.compile(file);
  };

  async store(buf: ArrayBuffer, filename: string) {
    console.log(`InMemoryStore: store ${buf.byteLength} bytes in ${filename}`);
    this.storage.set(filename, buf);
    return new File([buf], filename);
  };

}

// digest a file to a [32]Uint8Array
export async function digest(file: File, verbose = false): Promise<Uint8Array> {
  let t0 = performance.now();
  let sum = new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()));
  if (verbose) console.warn("SHA-256 digest of", file.name, `(${file.size} bytes)`, "took", performance.now() - t0, "ms.");
  return sum;
}
