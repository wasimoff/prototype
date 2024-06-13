// bare minimum worker, which simply sends the payload back
// mostly kept around as a template for new workers
// https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers
/// <reference lib="webworker" />

import { expose, workerReady } from "@/workerpool";

export class SimpleWorker {
  constructor(public name: string) { }

  // styled prefix for console logging with an orange background
  private get prefix() { return [ `%c SimpleWorker:${this.name} `, 'background: #00c4a7; color: black;' ]; }

  /** Return a string array in reverse order. */
  public reverse(array: string[]): { name: string, array: string[] } {
    console.log(...this.prefix, "received:", array);
    return { name: this.name, array: array.reverse() };
  };

  /**Super inefficient calculator for the `n`-th Fibonacci number. */
  public fib = (n: number): number => (n < 2) ? (n) : (this.fib(n-1) + this.fib(n-2));

  /** Say Hello! */
  public hello() { return `Hello from <${this.name}>!`; }

}

expose(SimpleWorker);
postMessage(workerReady);


/** Usage example:

import { SimpleWorker } from "@/worker/simple";
const simple = construct(new Worker(new URL("@/worker/simple", import.meta.url), { type: "module" }),
  SimpleWorker, "TrivialReally");

// post a string array to the SimpleWorker
async function post() {
  try {
    let worker = await simple;
    let message = [ "the current time", "is", new Date().toLocaleTimeString() ];
    let reply = await worker.reverse(message);
    terminal.info(`SimpleWorker <${reply.name}> replied: "${reply.array.join(" ")}"`);
  } catch (err) {
    terminal.error(`Failed to post message: ${(<Error>err).message}`);
  }
};

*/