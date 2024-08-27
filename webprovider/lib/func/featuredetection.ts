/** Check for the existence of a few key APIs. Some browsers or configurations
 * might disallow access to parts of these and the application won't work. */
export function CheckFeatures() {

  function check(expr: boolean, error: string): Error | undefined {
    return expr ? undefined : new MissingFeature(error);
  };

  let results = [

    // need OPFS for file storage
    // https://caniuse.com/native-filesystem-api
    check(("storage" in navigator && typeof navigator.storage.getDirectory === "function"), "Origin-Private FileSystem (OPFS) not available"),

    // need WebSocket / WebTransport for broker connection
    // https://caniuse.com/websockets, https://caniuse.com/webtransport
    check(("WebSocket" in window && typeof window.WebSocket.constructor === "function"), "WebSocket not available"),
    check(("WebTransport" in window && typeof window.WebTransport.constructor === "function"), "WebTransport not available"),

    // need WebAssembly support, obviously
    // https://caniuse.com/wasm
    check(("WebAssembly" in window && typeof window.WebAssembly.constructor === "function"), "WebAssembly not available"),

    // need Workers for multithreaded processing, SharedWebWorker for multi-tab pooling
    // https://caniuse.com/webworkers, https://caniuse.com/sharedworkers
    check(("Worker" in window && typeof window.Worker.constructor === "function"), "Web Workers not available"),
    check(("SharedWorker" in window && typeof window.SharedWorker.constructor === "function"), "Shared Web Workers not available"),

  ].filter(err => err !== undefined);

  if (results.length) {
    results.forEach(err => console.error(err));
    throw new Error("Prerequisites not met!\n" + results.join("\n"));
  };

};

class MissingFeature extends Error {
  constructor(message: string) {
    super(message);
    this.name = this.constructor.name;
  };
};
