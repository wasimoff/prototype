import { LRUCache } from "lru-cache";
import { MemoryFileSystem } from "./memory.ts";

const logprefix = [ "%c[ProviderStorage]", "color: purple;" ];

const twentyfourhours = 24*60*60*1000; // in milliseconds

/** ProviderStorage is an abstract interface to store and retrieve WebAssembly
 * executables and packed assets. It can for example be backed by a simple
 * in-memory cache or the Origin-Private Filesystem (OPFS) in browsers. */
export class ProviderStorage {

  // underlying filesystem implementation
  filesystem: ProviderStorageFileSystem;

  // base origin for the remote fetching
  public origin: string;

  public updates = new EventEmitter<{ added?: string[], removed?: string[] }>();

  // cache compiled webassembly modules
  private wasmCache = new LRUCache<string, WebAssembly.Module>({
    max: 10, ttl: twentyfourhours,
    fetchMethod: async (filename) => {
      let file = await this.getFile(filename);
      if (file === undefined) return undefined;
      return await WebAssembly.compile(await file.arrayBuffer());
    },
  });

  // cache zip archives for rootfs
  private zipCache = new LRUCache<string, ArrayBuffer>({
    max: 3, ttl: twentyfourhours,
    fetchMethod: async (filename) => {
      let file = await this.getFile(filename);
      if (file === undefined) return undefined;
      return await file.arrayBuffer();
    },
  });

  constructor(path: string, origin: string) {

    if (path === ":memory:") {
      this.filesystem = new MemoryFileSystem();
      console.debug(...logprefix, `opened storage with`, this.filesystem.constructor.name);
    } else {
      throw "OPFS not reimplemented yet."
      // this needs more work due to async initializer
      // OriginPrivateFileSystem.open(path).then(fs => this.filesystem = fs);
    };

    this.origin = origin;

  };

  // fetch a file from the backend
  private async fetchFile(filename: string): Promise<File | undefined> {

    // request the file from broker
    console.warn(...logprefix, `file ${filename} not found locally, fetch from broker`);
    let response = await fetch(`${this.origin}/storage/${filename}`);
    if (!response.ok) return undefined;

    // store fetched file to filesystem
    let buf = await response.arrayBuffer();
    let media = response.headers.get("content-type") || "";
    let name = response.headers.get("x-wasimoff-ref") || await getRef(buf);
    let file = new File([buf], name, { type: media });
    await this.filesystem.put(name, file);
    
    // emit event for broker
    this.updates.emit({ added: [ name ]});
    return file;

  };

  // TODO: emitting events for removed files requires shimming the FileSystem functions

  // either return a file from filesystem or attempt to fetch it remotely
  private async getFile(filename: string): Promise<File | undefined> {
    let file = await this.filesystem.get(filename);
    if (!file) file = await this.fetchFile(filename);
    return file;
  };

  /** Get a WebAssembly module compiled from a stored executable. */
  async getWasmModule(filename: string): Promise<WebAssembly.Module | undefined> {
    return this.wasmCache.fetch(filename);
  };

  /** Get a ZIP archive for rootfs usage. */
  async getZipArchive(filename: string): Promise<ArrayBuffer | undefined> {
    return this.zipCache.fetch(filename);
  };

}

/** ProviderStorageFileSystem is an underlying structure, which actually holds the
 * files. It minimally needs to support list, get, put and rm operations. */
export interface ProviderStorageFileSystem {

  /** Return the currently opened path. */
  readonly path: string;

  /** List all files in this Filesystem. */
  list(): Promise<string[]>;

  /** Get a specific file from Filesystem. */
  get(filename: string): Promise<File | undefined>;

  /** Save a new file to the Filesystem. */
  put(filename: string, file: File): Promise<File>;

  /** Remove a file from the Filesystem. */
  rm(filename: string): Promise<boolean>;

};

/** Return the SHA-256 digest of a file. This can be used to check for an exact match
 * without actually transferring the file's contents. */
export async function digest(buf: ArrayBuffer): Promise<Uint8Array> {
  if (crypto.subtle) return new Uint8Array(await crypto.subtle.digest("SHA-256", buf));
  else return new Uint8Array(32); // will always re-transfer
}

/** Check if a filename is a SHA256 content address aka. ref. */
export function isRef(filename: string): boolean {
  return filename.match(/^sha256:[0-9a-f]{64}$/i) !== null;
}

export async function getRef(buf: ArrayBuffer): Promise<string> {
  if (!crypto.subtle) throw "cannot compute digest in an insecure context";
  let hash = await digest(buf);
  let hex = [...hash].map(d => d.toString(16).padStart(2, "0")).join("");
  return `sha256:${hex}`;
}

/** Return a bytelength in human-readable unit. */
export function filesize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024**2) return `${(bytes/1024).toFixed(2)} KiB`;
  if (bytes < 1024**3) return `${(bytes/1024**2).toFixed(2)} MiB`;
  return `${(bytes/1024**3).toFixed(2)} GiB`;
};


// TODO: should move this to a generic location, but solves the problem for now
export class EventEmitter<T> {
  private listeners: ((message: T) => void)[] = [];

  on(listener: (message: T) => void): void {
    this.listeners.push(listener);
  }

  emit(message: T): void {
    this.listeners.forEach(listener => listener(message));
  }

  off(listener: (message: T) => void): void {
    const index = this.listeners.indexOf(listener);
    if (index !== -1) {
      this.listeners.splice(index, 1);
    }
  }
}
