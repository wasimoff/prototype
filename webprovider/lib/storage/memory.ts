import { ProviderStorageFileSystem } from "./index.ts";

const logprefix = [ "%c[MemoryFileSystem]", "color: purple;" ];

export class MemoryFileSystem implements ProviderStorageFileSystem {

  // always working in-memory, mimic the sqlite string
  readonly path = ":memory:";

  // just keep files in a map
  private storage = new Map<string, File>();

  // list all keys from the map
  async list(): Promise<string[]> {
    let list = [ ...this.storage.keys() ];
    console.debug(...logprefix, `has ${list.length} files:`, list);
    return list;
  };

  // return file from map
  async get(filename: string): Promise<File | undefined> {
    return this.storage.get(filename);
  };

  // store a new file in the map
  async put(filename: string, file: File): Promise<File> {
    console.debug(...logprefix, `store:`, file);
    this.storage.set(filename, file);
    return file;
  };

  // remove a file from map
  async rm(filename: string): Promise<boolean> {
    console.debug(...logprefix, `delete:`, filename);
    return this.storage.delete(filename);
  };

  // remove all files
  async prune(): Promise<string[]> {
    let list = [ ...this.storage.keys() ];
    this.storage.clear();
    return list;
  };

}
