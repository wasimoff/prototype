import { create, isMessage, Message as ProtoMessage } from "@bufbuild/protobuf";
import * as pb from "@wasimoff/proto/messages_pb.ts";
import { getRef, isRef } from "@wasimoff/storage/index.ts";
import { WasimoffProvider } from "./provider.ts";

// Handle incoming RemoteProcedureCalls on the Messenger iterable. Moved into a
// separate file for better readability and separation of concerns in a way.

export async function rpchandler(this: WasimoffProvider, request: ProtoMessage): Promise<ProtoMessage> {
  switch (true) {

    // execute wasi binary
    case isMessage(request, pb.ExecuteWasiRequestSchema): return <Promise<pb.ExecuteWasiResponse>>(async () => {

      // deconstruct the request
      let { info, task } = request;
      if (info === undefined || task === undefined || task.binary === undefined)
        throw "info, task and task.binary cannot be undefined";
      const taskid = `${info.jobID}/${info.index}`;

      // get or compile the webassembly module
      let wasm: WebAssembly.Module;
      if (task.binary.blob.length !== 0) {
        wasm = await WebAssembly.compile(task.binary.blob);
      } else if (task.binary.ref !== "") {
        if (this.storage === undefined) throw "cannot access storage yet";
        let m = await this.storage.getWasmModule(task.binary.ref);
        if (m === undefined) throw "binary not found in storage";
        else wasm = m;
      } else {
        throw "binary: neither blob nor ref were given";
      };

      console.log("WASM TASK:", wasm, task);

      // execute the module in a worker
      // TODO: handle task.rootfs and task.artifacts
      let run = await this.pool.run(taskid, {
        wasm: wasm,
        argv: task.args,
        envs: task.envs,
        stdin: task.stdin,
      });

      // send back the result
      return create(pb.ExecuteWasiResponseSchema, {
        result: {
          status: run.returncode,
          stdout: run.stdout,
          stderr: run.stderr,
          // TODO: artifacts
        },
      });
    })();

    // list files in storage
    case isMessage(request, pb.FileListingRequestSchema): return <Promise<pb.FileListingResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      const files = (await this.storage.lsf()).map(file => file.name);
      console.info("Sent list of available files to Broker.");
      return create(pb.FileListingResponseSchema, { files });
    })();

    // probe for a specific file in storage
    case isMessage(request, pb.FileProbeRequestSchema): return <Promise<pb.FileProbeResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      let ok = await this.storage.getFile(request.file) !== undefined;
      return create(pb.FileProbeResponseSchema, { ok });
    })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(request, pb.FileUploadRequestSchema): return <Promise<pb.FileUploadResponse>>(async () => {
      if (request.upload === undefined) throw "empty upload";
      if (this.storage === undefined) throw "cannot access storage yet";
      let { blob, /* media, */ ref } = request.upload;
      if (!isRef(ref)) {
        // overwrite name with computed digest
        ref = await getRef(new File([blob], ref));
      };
      await this.storage.store(blob, ref);
      return create(pb.FileUploadResponseSchema, { });
    })();

    default:
      throw "not implemented yet";

  };
};
