import { create, isMessage, Message as ProtoMessage } from "@bufbuild/protobuf";
import * as wasimoff from "@wasimoff/proto/v1/messages_pb.ts";
import { getRef, isRef } from "@wasimoff/storage/index.ts";
import { WasimoffProvider } from "./provider.ts";

// Handle incoming RemoteProcedureCalls on the Messenger iterable. Moved into a
// separate file for better readability and separation of concerns in a way.

export async function rpchandler(this: WasimoffProvider, request: ProtoMessage): Promise<ProtoMessage> {
  switch (true) {

    // execute a task
    case isMessage(request, wasimoff.Task_RequestSchema): return <Promise<wasimoff.Task_Response>>(async () => {

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
            return create(wasimoff.Task_ResponseSchema, {
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
            return create(wasimoff.Task_ResponseSchema, {
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
            return create(wasimoff.Task_ResponseSchema, {
              result: { case: "pyodide", value: {
                result: { case: "ok", value: {
                  pickle: run.pickle,
                  stdout: run.stdout,
                  stderr: run.stderr,
                  version: run.version,
                }},
              }},
            });

          } catch (err) {
            // format exceptions as WasiResponse.Error
            return create(wasimoff.Task_ResponseSchema, {
              result: {
                case: "error",
                value: String(err),
              },
            });
          };


      };
    })();

    // cancel a running task
    case isMessage(request, wasimoff.Task_CancelSchema): return <Promise<wasimoff.Task_Cancel>>(async () => {
      const { id, reason } = request;
      if (id !== undefined) {
        console.warn(...WasimoffProvider.logprefix, `cancelling task '${id}': ${reason}`);
        await this.pool.cancel(id);
      } else {
        throw "missing the task id to cancel!";
      };
      return request; // echo back
    })();

    // list files in storage
    case isMessage(request, wasimoff.FileListingRequestSchema): return <Promise<wasimoff.FileListingResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      const files = (await this.storage.filesystem.list());
      return create(wasimoff.FileListingResponseSchema, { files });
    })();

    // probe for a specific file in storage
    case isMessage(request, wasimoff.FileProbeRequestSchema): return <Promise<wasimoff.FileProbeResponse>>(async () => {
      if (this.storage === undefined) throw "cannot access storage yet";
      let ok = await this.storage.filesystem.get(request.file) !== undefined;
      return create(wasimoff.FileProbeResponseSchema, { ok });
    })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(request, wasimoff.FileUploadRequestSchema): return <Promise<wasimoff.FileUploadResponse>>(async () => {
      if (request.upload === undefined) throw "empty upload";
      if (this.storage === undefined) throw "cannot access storage yet";
      let { blob, media, ref } = request.upload;
      // overwrite name with computed digest
      if (!isRef(ref)) { ref = await getRef(blob); };
      await this.storage.filesystem.put(ref, new File([blob], ref, { type: media }));
      return create(wasimoff.FileUploadResponseSchema, { });
    })();

    default:
      throw "not implemented yet";

  };
};
