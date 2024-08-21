import { toBinary, fromBinary, toJsonString, fromJsonString } from "@bufbuild/protobuf";
import { EnvelopeSchema, Subprotocol, type Envelope, Envelope_MessageType } from "@/proto/messages_pb";
import { type Transport } from "./";
import { PushableAsyncIterable } from "@/fn/pushableasynciterable";
import { Signal } from "@/fn/utilities";

export class WebSocketTransport implements Transport {
  
  /** Connect using any known WebSocket subprotocol. */
  public static connect(url: string | URL): WebSocketTransport {
    let ws = new WebSocket(url, [
      // offer all known subprotocols on connection
      WebSocketTransport.provider_v1_protobuf,
      WebSocketTransport.provider_v1_json,
    ]);
    return new WebSocketTransport(ws);
  };

  /** Setup a Transport from an opened connection and wire up all the event listeners. */
  private constructor(private ws: WebSocket) {
    this.ws.binaryType = "arraybuffer";

    this.ws.addEventListener("open", () => {
      console.log(...prefixOpen, "connection established", { url: this.ws.url, protocol: this.ws.protocol });
      this.ready.resolve();
    });

    this.ws.addEventListener("error", (event) => {
      // per MDN: "fired when a connection [...] has been closed due to an error"
      console.error(...prefixErr, "connection closed due to an error", event);
    });

    this.ws.addEventListener("close", ({ code, reason, wasClean }) => {
      // TODO: implement reconnection handler without tearing everything down
      console.warn(...prefixWarn, `WebSocket connection closed:`, { code, reason, wasClean, url: this.ws.url });
      this.close(code, reason, wasClean);
    });

    this.ws.addEventListener("message", ({ data }) => {
      try {
        let envelope = this.unmarshal(data);
        if (debugging) console.debug(...prefixRx, envelope.sequence, Envelope_MessageType[envelope.type], envelope.payload, envelope.error);
        this.messages.push(envelope);
      } catch (err) {
        console.error(...prefixErr, err);
        this.messages.push(Promise.reject(err));
      };
    });

  };

  /** messages is an iterable of all incoming, already unmarshalled to Envelopes */
  public messages = new PushableAsyncIterable<Envelope>();

  /** send picks the correct codec depending on negotiated subprotocol and marshalls the envelope */
  public async send(envelope: Envelope): Promise<void> {
    this.closed.throwIfAborted();
    await this.ready.promise;
    if (debugging) console.debug(...prefixTx, envelope.sequence, Envelope_MessageType[envelope.type], envelope.payload, envelope.error);
    switch (this.ws.protocol) {

      case WebSocketTransport.provider_v1_protobuf:
        return this.ws.send(toBinary(EnvelopeSchema, envelope));

      case WebSocketTransport.provider_v1_json:
        return this.ws.send(toJsonString(EnvelopeSchema, envelope));

      default: // oops?
        let err = WebSocketTransport.Err.ProtocolViolation.Negotiation(this.ws.protocol);
        this.close(1002, err.message);
        throw err;
    };
  };

  /** unmarshal does just that and picks the correct codec based on negotiated subprotocol */
  private unmarshal(data: string | ArrayBuffer): Envelope {
    switch (this.ws.protocol) {

      case WebSocketTransport.provider_v1_protobuf:
        if (data instanceof ArrayBuffer)
          return fromBinary(EnvelopeSchema, new Uint8Array(data));
        else throw WebSocketTransport.Err.ProtocolViolation.MessageType("text", this.ws.protocol);

      case WebSocketTransport.provider_v1_json:
        if (typeof data === "string")
          return fromJsonString(EnvelopeSchema, data);
        else throw WebSocketTransport.Err.ProtocolViolation.MessageType("binary", this.ws.protocol);

      default: // oops?
        let err = WebSocketTransport.Err.ProtocolViolation.Negotiation(this.ws.protocol);
        this.close(1002, err.message);
        throw err;
    };
  }

  // signal to wait for readiness when sending
  private ready = Signal();

  // handle closure and cancellation
  private controller = new AbortController();
  public closed = this.controller.signal;

  public close(code: number = 1000, reason: string = "closed by user", wasClean: boolean = true) {
    // see https://www.rfc-editor.org/rfc/rfc6455.html#section-7.4 for defined status codes
    // but ws.close(code) can only be [ 1000, 3000..4999 ], so leave it blank below
    let err = new WebSocketTransport.Err.TransportClosed(code, reason, wasClean, this.ws.url);
    this.ws.close(undefined, reason);
    this.messages.close();
    this.ready.reject(err);
    this.controller.abort(err);
  };

};


export namespace WebSocketTransport {

  // provide shorthands for the subprotocols as strings
  export const provider_v1_protobuf = Subprotocol[Subprotocol.wasimoff_provider_v1_protobuf];
  export const provider_v1_json = Subprotocol[Subprotocol.wasimoff_provider_v1_json];

  // define possible error classes statically
  // extend Errors for custom error names
  export namespace Err {

    // the underlying connection was closed
    export class TransportClosed extends Error {
      constructor(public code: number, public reason: string, public wasClean: boolean, public url: string) {
        super(`WebSocket closed: ${JSON.stringify({ code, reason })}`);
        this.name = this.constructor.name;
      };
    };

    // unsupported protocol on the wire
    export class ProtocolViolation extends Error {
      constructor(message: string, public protocol: string) {
        super(`${message}: ${protocol}`);
        this.name = this.constructor.name;
      };
      static Negotiation(p: string) {
        return new ProtocolViolation("unsupported protocol", p);
      };
      static MessageType(t: string, p: string) {
        return new ProtocolViolation(`wrong message type ${t} for protocol`, p);
      };
    };

  };

};

// enable console.logs in the "hot" path (tx/rx)?
const debugging = true;

// pretty console logging prefixes
const prefixOpen = [ "%c WebSocketTransport %c open ", "background: #333; color: white;", "background: greenyellow;" ];
const prefixRx   = [ "%c WebSocketTransport %c « Rx %c %s ", "background: #333; color: white;", "background: skyblue;", "background: #ccc;" ];
const prefixTx   = [ "%c WebSocketTransport %c Tx » %c %s ", "background: #333; color: white;", "background: greenyellow;", "background: #ccc;" ];
const prefixErr  = [ "%c WebSocketTransport %c Error ", "background: #333; color: white;", "background: firebrick; color: white;" ];
const prefixWarn = [ "%c WebSocketTransport %c Warning ", "background: #333; color: white;", "background: goldenrod;" ];

