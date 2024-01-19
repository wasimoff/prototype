/** Check for the existence of a few key APIs. Some browsers or configurations
 * might disallow access to parts of these and the application won't work. */
export function checkPrerequisites() {
  let results = [

    // need opfs for file storage
    check(("storage" in navigator && typeof navigator.storage.getDirectory === "function"), "Origin-Private FileSystem (OPFS) not available"),

    // need webtransport for broker connection
    check(("WebTransport" in window && typeof window.WebTransport.constructor === "function"), "WebTransport not available"),

    // need webassembly support
    check(("WebAssembly" in window && typeof window.WebAssembly.constructor === "function"), "WebAssembly not available"),

    // need webtransport for broker connection
    check(("Worker" in window && typeof window.Worker.constructor === "function"), "Web Workers not available"),

  ].filter(err => err !== undefined);
  if (results.length) {
    results.forEach(err => console.error(err));
    throw new Error("Prerequisites not met!\n" + results.join("\n"));
  };
};

function check(expr: boolean, error: string): Error | undefined {
  return expr ? undefined : new Error(error);
};