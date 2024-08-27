/** A simple asynchronous "FIFO" queue class. */
// Heavily inspired by github.com/NicoAdrian/async-fifo-queue, but adapted
// with generic types for Typescript and a simpler unbounded queue.
// Original Copyright (c) 2020 NicoAdrian (MIT Licensed)
export class Queue<T> {
  // private prefix = (op: string) => [ `%c QUEUE ${op} `, "background: lime; color: black;" ];

  // resolve functions of waiting getters
  private getters: ((item: T) => void)[] = [];

  // the queued items
  private items: T[] = [];

  // yield an item from the queue
  async get(): Promise<T> {
    // console.debug(...this.prefix("-GET"), { items: this.items.length, waiting: this.getters.length });
    // if the queue is empty, append ourselves as a waiting promise
    if (this.items.length === 0) {
      return await new Promise<T>(r => this.getters.push(r));
    }
    // otherwise yield an element immediately
    return this.items.shift()!;
  };

  // put an item into the queue
  async put(item: T): Promise<void> {
    // console.debug(...this.prefix("+PUT"), { items: this.items.length, waiting: this.getters.length });
    // if there are getters waiting, resolve the first
    if (this.getters.length > 0) {
      return this.getters.shift()!(item);
    }
    // otherwise append to queue
    this.items.push(item);
  };

};
