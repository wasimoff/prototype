import { create, isMessage, Message as ProtoMessage } from "@bufbuild/protobuf";
import * as pb from "@wasimoff/proto/messages_pb.ts";
import { getRef, isRef } from "@wasimoff/storage/index.ts";
import { WasimoffProvider } from "./provider.ts";

// Handle incoming RemoteProcedureCalls on the Messenger iterable. Moved into a
// separate file for better readability and separation of concerns in a way.

export async function rpchandler(this: WasimoffProvider, request: ProtoMessage): Promise<ProtoMessage> {
  switch (true) {

    // execute a task
    case isMessage(request, pb.Task_RequestSchema): return <Promise<pb.Task_Response>>(async () => {

      // deconstruct the request and check type
      let { info, parameters } = request;
      if (info === undefined || parameters === undefined)
        throw "info and parameters cannot be undefined";
      if (parameters.case === undefined || !["wasip1", "pyodide"].includes(parameters.case))
        throw "unknown task format";

      // inner switch by type
      switch (parameters.case) {

        // -------------------------------------------------------------------
        case "wasip1":
          const task = parameters.value;
          if (task.binary === undefined)
            throw "wasip1.binary cannot be undefined";
    
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
            throw new Error("binary: neither blob nor ref were given");
          };
    
          // get rootfs archive
          let rootfs: Uint8Array | undefined;
          if (task.rootfs !== undefined) {
            if (task.rootfs.blob.length !== 0) {
              rootfs = task.rootfs.blob;
            } else if (task.rootfs.ref !== "") {
              if (this.storage === undefined) throw "cannot access storage yet";
              let z = await this.storage.getZipArchive(task.rootfs.ref);
              if (z === undefined) throw "zip not found in storage";
              else rootfs = new Uint8Array(z);
            } else {
              throw new Error("rootfs: neither blob nor ref were given");
            }
          }
    
          console.debug("%c[RPCHandler]", "color: orange;", task);
    
          try {
            // execute the module in a worker
            let run = await this.pool.runWasip1(info.id, {
              wasm: wasm,
              argv: task.args,
              envs: task.envs,
              stdin: task.stdin,
              rootfs: rootfs,
              artifacts: task.artifacts,
            });
            // send back the result
            return create(pb.Task_ResponseSchema, {
              result: {
                case: "wasip1",
                value: {
                  result: {
                    case: "ok",
                    value: {
                      status: run.returncode,
                      stdout: run.stdout,
                      stderr: run.stderr,
                      artifacts: run.artifacts ? { blob: run.artifacts } : undefined,
                    }
                  }
                }
              }
            });
          } catch (err) {
            // format exceptions as WasiResponse.Error
            return create(pb.Task_ResponseSchema, {
              result: {
                case: "error",
                value: String(err),
              },
            });
          };


        // -------------------------------------------------------------------
        case "pyodide":

          const pytask = parameters.value;
          if (pytask.script === undefined)
            throw "pyodide.script cannot be undefined";
    
          console.debug("%c[RPCHandler]", "color: orange;", pytask);
          try {

            let run = await this.pool.runPyodide(info.id, pytask);
            return create(pb.Task_ResponseSchema, {
              result: { case: "pyodide", value: {
                result: { case: "ok", value: {
                  pickle: run.pickle,
                  stdout: run.stdout,
                  stderr: run.stderr,
                }},
              }},
            });

          } catch (err) {
            // format exceptions as WasiResponse.Error
            return create(pb.Task_ResponseSchema, {
              result: {
                case: "error",
                value: String(err),
              },
            });
          };


      };
    })();

    // list files in storage
    case isMessage(request, pb.FileListingRequestSchema): return <Promise<pb.FileListingResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      const files = (await this.storage.filesystem.list());
      return create(pb.FileListingResponseSchema, { files });
    })();

    // probe for a specific file in storage
    case isMessage(request, pb.FileProbeRequestSchema): return <Promise<pb.FileProbeResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      let ok = await this.storage.filesystem.get(request.file) !== undefined;
      return create(pb.FileProbeResponseSchema, { ok });
    })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(request, pb.FileUploadRequestSchema): return <Promise<pb.FileUploadResponse>>(async () => {
      if (request.upload === undefined) throw "empty upload";
      if (this.storage === undefined) throw "cannot access storage yet";
      let { blob, media, ref } = request.upload;
      // overwrite name with computed digest
      if (!isRef(ref)) { ref = await getRef(blob); };
      await this.storage.filesystem.put(ref, new File([blob], ref, { type: media }));
      return create(pb.FileUploadResponseSchema, { });
    })();

    default:
      throw "not implemented yet";

  };
};
