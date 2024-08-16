import { type Request, type Response, type Event, type Envelope } from "@/proto/messages_pb";

/** Transport abstracts away an underlying network connection and marshalling
 * protocol to the Broker or another resource consumer. On the surface, it is a
 * simple interface to receive and send messages but it does not handle any
 * Request-Response semantics. It is possible to reconnect or migrate connections
 * transparently without breaking the message iterator. */
export interface Transport {

  // receive ordered messages from an iterable
  messages: AsyncIterable<Envelope>;

  // send messages with a simple function
  send: (envelope: Envelope) => Promise<void>;

  // signal a closed connection with an AbortController internally
  closed: AbortSignal;
  close: () => void;

}

export { WebSocketTransport } from "./websocket";
export { Messenger } from "./messenger";



// --------------------------------------------------------------------------------
// old transport index.ts below

import * as MessagePack from "@msgpack/msgpack";

/** An interface, which implements a connection to the Broker with various functions. */
export interface IBrokerTransport {

  // overall status of this transport
  closed: Promise<any>;
  close: () => Promise<void>;

  // a stream of rpc requests that should be handled
  rpc: RPCServer;

  // a bidirectional stream for control messages
  messages: AsyncChannel<unknown>;

}

// re-export the implemented transports
export { WebTransportBroker } from "./webtransport_old";

//? +----------------------------------------------------------+
//? | various types that must be implemented by the transports |
//? +----------------------------------------------------------+

/** The header of a Go `net/rpc` RPC request. */
export type NetRPCRequestHeader = { ServiceMethod: string, Seq: BigInt };

/** The header of a Go `net/rpc` RPC response. */
export type NetRPCResponseHeader = NetRPCRequestHeader & { Error?: string };

/** An RPC decoder is an async generator of RPC request information. */
export type NetRPCDecoder = AsyncGenerator<RPCRequestInfo, void, undefined>;

/** An RPC encoder takes responses to write and can be closed. */
export type NetRPCEncoder = { next: RPCResponder, close: () => Promise<void> };

/** The fields of an RPC request used internally. */
export type RPCRequestInfo = { method: string, seq: BigInt, body: any, error?: string };

/** A function that must be called with an async function to handle the RPC request. */
export type RPCRequest = (handler: (method: string, body: any) => Promise<any>) => Promise<void>;

/** Signature of a function that encodes and sends net/rpc-compatible responses to the requester. */
export type RPCResponder = (response: RPCRequestInfo) => Promise<void>;

/** An async generator of `RPCRequest`s to be handled. */
export type RPCServer = AsyncGenerator<RPCRequest, void, undefined>;

//? +-------------------------------------------------------------------+
//? | a generic wrapper which makes bidirectional streams easier to use |
//? +-------------------------------------------------------------------+

// type for an asymmetric channel
export interface AsyncChannel<Value> {
  channel: AsyncGenerator<Value, void, undefined>;
  send: (value: Value) => Promise<void>;
  close: () => Promise<void>;
};

export class MessagePackChannel<Message = any> implements AsyncChannel<Message> {

  /** `channel` asynchronously receives incoming messages */
  public channel: AsyncGenerator<Message, void, undefined>;

  /** `writer` is the locked writable stream for the encoder */
  private writer: WritableStreamDefaultWriter<Uint8Array>;

  /** Locks a bidirectional stream to use it as a channel for any MessagePack messages. */
  constructor(private stream: WebTransportBidirectionalStream) {

    // the receive channel is simply the messagepack decoder
    //! it is important that the generator releases the lock on the stream on return()
    this.channel = MessagePack.decodeMultiStream(stream.readable) as typeof this.channel;

    // the encoder needs to lock the writer and keep it around so it can be released
    this.writer = stream.writable.getWriter();

  };

  /** `send` can be used to send messages asynchronously */
  async send(message: Message) {
    await this.writer.ready;
    let chunk = MessagePack.encode(message, { useBigInt64: true });
    return this.writer.write(chunk);
  };

  /** `close` tries to gracefully close the channel */
  async close() {

    // close both directional streams in parallel
    await Promise.allSettled([

      // release lock on the generator, then close the readable
      this.channel.throw("").then(() => this.stream.readable.cancel())
      .then(() => console.error("MessagePackChannel `channel` closed")),

      // we're holding the lock on writer, so we can close it directly
      this.writer.close().then(() => this.writer.releaseLock())
      .then(() => console.error("MessagePackChannel `writer` closed")),

    ]);
    
  };

}