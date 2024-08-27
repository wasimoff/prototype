/** Collected events with timestamps. */
type Event<T> = { time: T, label: string };

/** An exported Trace with start time and events in microsecond unix epochs. */
// export type ExportedTrace = { start: BigInt, events: Event<BigInt>[] };
export type ExportedTrace = Event<BigInt>[];

/** A simple class to log timestamps at certain points of the execution. */
export class Trace {

  constructor(label?: string) {
    if (label !== undefined) this.now(label);
  };

  // time origin and starting time offset in milliseconds
  private readonly origin = performance.timeOrigin;
  // private readonly t0: number = performance.now();

  // calculate the unix epoch
  private epoch(t: number) { return this.origin + t; };

  // calculate unix epochs in microseconds
  private unixmicro(t: number) {
    return BigInt(this.epoch(t).toPrecision(16).replace(".", ""));
  };

  // format an ISO string with microseconds
  // private isomicro(t: number) {
  //   let epoch = this.epoch(t);
  //   let iso = new Date(epoch).toISOString();
  //   console.error("EVENT", epoch, iso, (epoch % 1).toPrecision(3));
  //   return iso.replace("Z", (epoch % 1).toPrecision(3).substring(2, 5) + "Z");
  // }

  // collected events with timestamps
  private events: Event<number>[] = [ ];

  /** Add a new event with a string label and optional data. */
  public async now(label: string) {
    this.events.push({ time: performance.now(), label });
  };

  /** Export the collected events and calculate deltas. */
  public async export(): Promise<ExportedTrace> {
    return this.events.map(ev => ({
      label: ev.label,
      time: this.unixmicro(ev.time),
    }));
  };

};
