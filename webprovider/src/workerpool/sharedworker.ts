/// <reference lib="webworker" />
declare var self: SharedWorkerGlobalScope;
export {};


import { type WasiTaskExecution } from "./wasiworker";
import { WasiWorkerPool } from "./workerpool";
import { expose, workerReady } from "@/workerpool";


// ---- initialize shared worker for wasimoff ---- //
const logprefix = [ "%c wasimoff ", "background-color: violet;" ];

// time of inception
const spawned = new Date().toLocaleString();
console.log(...logprefix, "spawned at", spawned);

const pool = new WasiWorkerPool(16);

export const SharedWasimoff = {

  async fill() {
    return pool.fill();
  },

  async run(id: string, task: WasiTaskExecution) {
    console.log(...logprefix, "RUN", id, task);
    return pool.run(id, task);
  },

  async race(n: number, task: WasiTaskExecution) {
    if (!(task.wasm instanceof WebAssembly.Module))
      task.wasm = await WebAssembly.compile(task.wasm);
    let t0 = performance.now();
    let promises = Array(n).fill(null).map((_, i) => this.run(`${i}`, task));
    await Promise.all(promises);
    let t1 = performance.now() - t0;
    console.warn(...logprefix, "race finished after", t1);
    return t1;
  },

}

self.addEventListener("connect", conn => {
  console.log(...logprefix, "New Connection", conn.origin);
  let port = conn.ports[0];
  port.addEventListener("message", ({ data }) => {
    console.info(data);
    switch (data.type) {
      case "fill": (async () => {
        await SharedWasimoff.fill();
        port.postMessage({ type: "fill", payload: pool.workers });
      })(); break;

      case "race": (async () => {
        let { n, task } = data as { n: number, task: WasiTaskExecution };
        let race = await SharedWasimoff.race(n, task);
        port.postMessage({ type: "race", payload: race });
      })(); break;
    }
  });
  // port.start();
  expose(SharedWasimoff, port);
  port.postMessage(workerReady);
});


// // keep references to opened tabs and remove them after timeout
// class Tabulator {

//   public tabs: TabConnection[] = [];

//   private tabCounter = 1;
//   public add(port: MessagePort) {
//     let connection = new TabConnection(this.tabCounter++, port);
//     connection.context.addEventListener("abort", () => this.remove(port));
//     this.tabs.push(connection);
//     tabulator.broadcast("tabs", tabulator.serialize());
//     return connection;
//   };

//   public remove(id: number | MessagePort) {
//     // remove from the tabs list
//     let removed = this.tabs.splice(this.tabs.findIndex(tab => {
//       if (typeof id === "number") return tab.id === id;
//       else return tab.port === id;
//     }), 1);
//     // if there was a removed match, make sure it's aborted
//     if (removed.length) removed[0].close();
//     tabulator.broadcast("tabs", tabulator.serialize());
//   };

//   public serialize(): { [id: number]: string} {
//     return this.tabs.reduce((tabs, t) => Object.assign(tabs, { [t.id]: t.openedAt.toLocaleString() }), { });
//   };

//   // TODO: replace with BroadcastChannel API
//   public broadcast(type: string, payload: any) {
//     for (let tab of this.tabs) tab.post(type, payload);
//   };

// }
// const tabulator = new Tabulator();


// // track a tab connection, with a watchdog timer to clean up stale ports
// class TabConnection {

//   public constructor(
//     public id: number,
//     public port: MessagePort,
//     private timeout = 2000
//   ) {
//     // add keepalive listener for watchdog
//     this.port.addEventListener("message", ({ data }) => {
//       if (data === "keepalive") this.keepalive();
//     });
//   };

//   // time of inception
//   public openedAt: Date = new Date();

//   // context to abort listeners and close port
//   private controller = new AbortController();
//   public context = this.controller.signal;
//   public close() { return this.controller.abort(); };
  
//   // watchdog timer, which aborts the context on timeout
//   //! the watchdog is "armed" only after the first keepalive message is received
//   private timer?: ReturnType<typeof setTimeout>;
//   public keepalive() {
//     if (this.timer) clearTimeout(this.timer);
//     this.timer = setTimeout(() => {
//       this.controller.abort();
//       this.port.close();
//       console.error(...logprefix, `Tab connection ${this.id} timed out!`);
//     }, this.timeout);
//   };

//   // TODO: typed posting or rather use wrapped comlinks for <MessagePort>?
//   public post(type: string, payload: any) {
//     this.port.postMessage({ type, payload });
//   };

// }



// for each opened connection to this SharedWorker ...
// self.onconnect = (ev) => {

//   const tab = tabulator.add(ev.ports[0]);
//   console.log(...logprefix, "New Tab:", tab);

//   // handle incoming messages
//   tab.port.addEventListener("message", ({ data }) => {
//     switch (data) {

//       case "keepalive":
//         tab.keepalive();
//         break;

//       case "list_tabs":
//         tab.keepalive();
//         tab.post("tabs", tabulator.serialize());
//         break;

//       case "close":
//         tab.close();
//         tabulator.broadcast("tabs", tabulator.serialize());
//         break;

//       default:
//         break;
//     };
//   }, { signal: tab.context });
//   tab.port.start();

// };
