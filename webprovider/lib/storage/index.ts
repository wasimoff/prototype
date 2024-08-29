/** ProviderStorage is an abstract interface to store and retrieve WebAssembly
 * executables and packed assets. It can for example be backed by a simple
 * in-memory Map or the Origin-Private Filesystem (OPFS) in browsers. */
export interface ProviderStorage {

  /** Return the currently opened path. */
  readonly path: string;

  /** List all files in Storage. */
  lsf(): Promise<File[]>;

  /** Save a new file to the Storage. */
  store(buf: ArrayBuffer, filename: string): Promise<File>;

  /** Remove a file from the Storage. */
  rm(filename: string): Promise<boolean>;

  /** Remove all files from the Storage. */
  prune(): Promise<string[]>;

  /** Get file contents as an ArrayBuffer. */
  getBuffer(filename: string): Promise<ArrayBuffer | undefined>;

  /** Get a WebAssembly module compiled from a stored executable. */
  getWasmModule(filename: string): Promise<WebAssembly.Module | undefined>;

}

// TODO: select with "memory://" or "opfs://directory"
// new URL("opfs://wasm").pathname.replace(/^\/\/+/, "/")
export { InMemoryStorage } from "./memory.ts";
export { OpfsStorage } from "./opfs.ts";

/** Return the SHA-256 digest of a file. This can be used to check for an exact match
 * without actually transferring the file's contents. */
export async function digest(file: File): Promise<Uint8Array> {
  if (crypto.subtle) return new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()));
  else return new Uint8Array(32); // will always re-transfer
}