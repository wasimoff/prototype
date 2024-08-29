import { create, isMessage, Message as ProtoMessage } from "@bufbuild/protobuf";
import * as pb from "@wasimoff/proto/messages_pb.ts";
import { digest } from "@wasimoff/storage/index.ts";
import { WasimoffProvider } from "./provider.ts";

// Handle incoming RemoteProcedureCalls on the Messenger iterable. Moved into a
// separate file for better readability and separation of concerns in a way.

export async function rpchandler(this: WasimoffProvider, request: ProtoMessage): Promise<ProtoMessage> {
  switch (true) {

    // execute wasi binary
    case isMessage(request, pb.ExecuteWasiArgsSchema): return <Promise<pb.ExecuteWasiResult>>(async () => {
      let id = request.task !== undefined ? `${request.task.id}/${request.task.index}` : "unknown";
      if (request.binary === undefined || request.binary.binary.case === "raw" || request.binary.binary.value === undefined) {
        throw "raw binary not implemented yet";
      }
      let filename = request.binary.binary.value!;
      let binary = await this.storage!.getWasmModule(filename);
      if (binary === undefined) throw "module not found";
      let run = await this.pool.run(id, {
        wasm: binary,
        argv: [filename, ...request.args],
        envs: request.envs,
        stdin: request.stdin,
      });
      return create(pb.ExecuteWasiResultSchema, {
        status: run.returncode,
        // TODO: we've just decoded the output in run!
        stdout: new TextEncoder().encode(run.stdout),
        stderr: new TextEncoder().encode(run.stderr),
      });
    })();

    // list files in storage
    case isMessage(request, pb.FileListingArgsSchema): return <Promise<pb.FileListingResult>>(async () => {
      let files = await Promise.all((await this.storage!.lsf()).map(async file => ({
        filename: file.name,
        length: BigInt(file.size),
        // epoch: BigInt(file.lastModified),
        hash: await digest(file),
      })));
      console.info("Sent list of available files to Broker.");
      return create(pb.FileListingResultSchema, { files });
    })();

    // probe for a specific file in storage
    case isMessage(request, pb.FileProbeArgsSchema): return <Promise<pb.FileProbeResult>>(async () => {
      let ok = await (async () => {
        // expect a normal file sans the bytes
        const { filename, hash, length } = request.stat!;
        // find the file by filename
        let file = (await this.storage!.lsf()).find(f => f.name === filename);
        if (file === undefined) return false;
        // check the filesize
        if (file.size !== Number(length)) return false;
        // calculate the sha256 digest, if file exists
        if (hash.byteLength !== 32) throw new Error("hash length must be 32 bytes");
        let sum = await digest(file);
        if (sum.byteLength !== 32) throw new Error("sha-256 digest has an unexpected length");
        // compare the hashes
        for (let i in sum) if (sum[i] !== hash[i]) return false;
        // file exists and hashes match
        return true;
      })();
      return create(pb.FileProbeResultSchema, { ok });
    })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(request, pb.FileUploadArgsSchema): return <Promise<pb.FileUploadResult>>(async () => {
      const { filename, /* hash, epoch */ } = request.stat!;
      await this.storage!.store(request.file, filename);
      return create(pb.FileUploadResultSchema, { ok: true });
    })();

    default:
      throw "not implemented yet";

  };
};