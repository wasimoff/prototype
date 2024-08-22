import { Envelope_MessageType as MessageType, EnvelopeSchema, file_messages } from "@/proto/messages_pb";
import { create, createRegistry, toBinary, type Message } from "@bufbuild/protobuf";
import { type Transport } from ".";
import { PushableAsyncIterable } from "@/fn/pushableasynciterable";
import { AnySchema, anyUnpack, type Any } from "@bufbuild/protobuf/wkt";


/** This interface is not technically needed. It's just there to
 * remind me to keep the Messenger API simple. */
interface MessengerInterface {

  // remote procedure calls
  requests: AsyncIterable<RemoteProcedureCall>;
  sendRequest: (r: Message) => Promise<Result>;

  // event messages
  events: AsyncIterable<Message>;
  sendEvent: (event: Message) => Promise<void>;

  // signal a closed transport
  closed: AbortSignal;
  close: () => void;

}

/** A remote procedure call is emitted by the AsyncIterator and must be called with an async handler,
 * which receives the Request object and produces a Result. If the handler throws, the caught error is
 * sent back to the caller automatically. */
export type RemoteProcedureCall = (handler: (request: Message) => Promise<Message>) => Promise<void>;


/** MessengerInterface wraps around some Transport, which could be a WebSocket,
 * WebTransport, direct WebRTC or really any other bidirectional stream inside,
 * and provides the handling of remote procedure calls (making sure that each
 * Request receives a Response etc.).
 * TODO: this should probably be extended in the future to be able to wrap multiple
 * Transports and present only a single interface to the Provider app. */
export class Messenger implements MessengerInterface {

  constructor(private transport: Transport) {
    this.switchboard();
  }

  private readonly registry = createRegistry(file_messages);

  private async switchboard() {
    for await (const m of this.transport.messages) {
      switch (m.type) {

        case MessageType.Request:
          // construct a RemoteProcedureCall that will send a response when it's done
          //? careful not to await the call itself here, otherwise stream is blocked
          this.requests.push(async (handler) => {
            // prepare a response envelope
            let r = create(EnvelopeSchema, { type: MessageType.Response, sequence: m.sequence });
            try {
              // unpack the any payload
              let request = this.unpack(m.payload);
              // call the handler and marshal the result
              let result = await handler(request);
              r.payload = this.pack(result);
            } catch (err) {
              // oops: report the error to the client
              r.error = String(err)
              r.payload = undefined
            } finally {
              // send whatever we could gather back
              await this.transport.send(r);
            };
          });
          break;

        case MessageType.Response:
          // find a pending request and resolve it; cleanup is done in sendRequest
          let pending = this.pending.get(m.sequence);
          if (m.error) {
            pending?.(new Error(m.error));
          } else {
            let response = this.unpack(m.payload);
            pending?.(response);
          };
          break;

        case MessageType.Event:
          // push the event to the iterable
          let e = anyUnpack(m.payload!, this.registry);
          this.events.push(e!);
          break;
      
        default:
          // empty message or unknown type
          console.warn("received a malformed letter:", m.sequence, m.type);
          break;

      }; // switch
    }; // for await

    // if we ever land here, the iteration failed; close the interface
    this.close(new Error("iterator exited"));
  };

  requests = new PushableAsyncIterable<RemoteProcedureCall>;

  private requestSequence = 0n;
  private pending = new Map<BigInt, (r: Result) => void>();
  async sendRequest(request: Message): Promise<Result> {
    // TODO: caution, Provider->Broker requests are not properly tested yet
    // get the next sequence number
    let sequence = this.requestSequence++;
    //create and register a promise for the pending request
    const result = new Promise<Result>(r => this.pending.set(sequence, r));
    try {
      // actually envelope the request and send it off
      await this.transport.send(create(EnvelopeSchema, {
        sequence, type: MessageType.Request, payload: this.pack(request),
      }));
      // await the result, so the finally doesn't run until it's done
      return await result;
    } finally {
      // clean up the pending promise
      this.pending.delete(sequence);
    }
  };

  events = new PushableAsyncIterable<Message>;

  private eventSequence = 0n;
  async sendEvent(event: Message): Promise<void> {
    // envelope the event and send it off
    return this.transport.send(create(EnvelopeSchema, {
      sequence: this.eventSequence++,
      type: MessageType.Event,
      payload: this.pack(event),
    }));
  };

  // handle closure and cancellation
  private controller = new AbortController();
  public closed = this.controller.signal;

  public close(reason: any = new Error("closed by user")) {
    // close iterables
    this.events.close();
    this.requests.close();
    // cancel pending requests
    this.pending.forEach(r => r(Promise.reject(reason) as any)); // TODO: type error
    this.pending.clear();
    // abort the controller
    this.controller.abort(reason);
  };


  private pack(m: Message): Any {
    let schema = this.registry.getMessage(m.$typeName);
    if (schema === undefined) throw "unknown message type";
    let into = create(AnySchema, {
      typeUrl: `wasimoff/${m.$typeName}`,
      value: toBinary(schema, m),
    });
    return into;
  };

  private unpack(p: Any | undefined): Message {
    if (p === undefined) throw "cannot unpack empty payload";
    let message = anyUnpack(p, this.registry);
    if (message === undefined) throw "unknown payload type";
    return message;
  };

}

export type Result = Error | PromiseLike<Error> | Message | PromiseLike<Message>;