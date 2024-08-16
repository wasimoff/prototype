import { EnvelopeSchema, ResponseSchema, type Event, type Request, type Response } from "@/proto/messages_pb";
import { create } from "@bufbuild/protobuf";
import { type Transport } from ".";
import { PushableAsyncIterable } from "@/fn/pushableasynciterable";


/** This interface is not technically needed. It's just there to
 * remind me to keep the Messenger API simple. */
interface MessengerInterface {

  // remote procedure calls
  requests: AsyncIterable<RemoteProcedureCall>;
  sendRequest: (r: Request) => Promise<Response>;

  // event messages
  events: AsyncIterable<Event>;
  sendEvent: (event: Event) => Promise<void>;

  // signal a closed transport
  closed: AbortSignal;
  close: () => void;

}

/** A remote procedure call is emitted by the AsyncIterator and must be called with an async handler,
 * which receives the Request object and produces a Result. If the handler throws, the caught error is
 * sent back to the caller automatically. */
export type RemoteProcedureCall = (handler: (request: Request) => Promise<Response>) => Promise<void>;


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

  private async switchboard() {
    for await (const { sequence, message } of this.transport.messages) {
      switch (message.case) {

        case "request":
          // construct a RemoteProcedureCall that will send a response when it's done
          //? careful not to await the call itself here, otherwise stream is blocked
          this.requests.push(async (handler) => {
            // prepoluate the response with an error if everything fails
            let response = create(ResponseSchema, { error: "failed to execute rpc handler" });
            try {
              // happy path: encode the result
              response = await handler(message.value);
            } catch (err) {
              // oops: report the error to the client
              response = create(ResponseSchema, { error: String(err) });
            } finally {
              // send whatever we could gather back
              await this.transport.send(create(EnvelopeSchema, {
                sequence, message: { case: "response", value: response },
              }));
            };
          });
          break;

        case "response":
          // find a pending request and resolve it; cleanup is done in sendRequest
          this.pending.get(sequence)?.(message.value);
          break;

        case "event":
          // push the event to the iterable
          this.events.push(message.value);
          break;
      
        default:
          // empty message or unknown type
          console.warn("received a malformed letter:", sequence, message);
          break;

      }; // switch
    }; // for await

    // if we ever land here, the iteration failed; close the interface
    this.close(new Error("iterator exited"));
  };

  requests = new PushableAsyncIterable<RemoteProcedureCall>;

  private requestSequence = 0n;
  private pending = new Map<BigInt, (r: Response | PromiseLike<Response>) => void>();
  async sendRequest(request: Request): Promise<Response> {
    // TODO: caution, Provider->Broker requests are not properly tested yet
    // get the next sequence number
    let sequence = this.requestSequence++;
    //create and register a promise for the pending request
    const result = new Promise<Response>(r => this.pending.set(sequence, r));
    try {
      // actually envelope the request and send it off
      await this.transport.send(create(EnvelopeSchema, {
        sequence, message: { case: "request", value: request },
      }));
      // await the result, so the finally doesn't run until it's done
      return await result;
    } finally {
      // clean up the pending promise
      this.pending.delete(sequence);
    }
  };

  events = new PushableAsyncIterable<Event>;

  private eventSequence = 0n;
  async sendEvent(event: Event): Promise<void> {
    // envelope the event and send it off
    return this.transport.send(create(EnvelopeSchema, {
      sequence: this.eventSequence++,
      message: { case: "event", value: event },
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
    this.pending.forEach(r => r(Promise.reject(reason)));
    this.pending.clear();
    // abort the controller
    this.controller.abort(reason);
  };

}

import { WebSocketTransport } from "./websocket";
import { EventSchema, RequestSchema } from "@/proto/messages_pb";
try {

  let wst = WebSocketTransport.connect("ws://localhost:4080/messagesock");
  let msg = new Messenger(wst);
  (async () => {
    
    // send an event
    await msg.sendEvent(create(EventSchema, { event: {
      case: "providerResources",
      value: {
        nmax: navigator.hardwareConcurrency,
        tasks: 0,
      },
    }}));

    try {
      let response = await msg.sendRequest(create(RequestSchema, {
      request: {
        case: "fileListingArgs",
        value: { },
      },
    }));
    console.log("broker response:", response);
  } catch (err) {
    console.warn("broker request error:", err);
  };
  
  for await (let rpc of msg.requests) {
    rpc(async request => {
      return create(ResponseSchema, {
        error: `not implemented yet: ${request.request.case}`
      });
    });
  };
  
})();

} catch (err) {
  console.log("oops:", err);
}