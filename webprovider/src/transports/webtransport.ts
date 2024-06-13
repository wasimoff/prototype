import * as MessagePack from "@msgpack/msgpack";
import { pairs, next } from "@/fn/utilities";
import type { NetRPCDecoder, NetRPCEncoder, NetRPCRequestHeader, NetRPCResponseHeader, RPCServer } from "@/transports";
import { MessagePackChannel, BrokerTransport } from "@/transports";

//? +--------------------------------------------------------------+
//? | Implement a Broker transport over a WebTransport connection. |
//? +--------------------------------------------------------------+

/** `WebTransportBroker` implements a WebTransport connection to the Broker, on
 * which there is an asymmetric channel for control messages and an async generator
 * of received RPC requests. Use `WebTransportBroker.connect()` to instantiate. */
export class WebTransportBroker implements BrokerTransport {

  private constructor(

    /** The underlying [`WebTransport`](https://developer.mozilla.org/docs/Web/API/WebTransport) connection. */
    public transport: WebTransport,

    /** Promise that is resolved or rejected when the transport is closed. */
    public closed: Promise<WebTransportCloseInfo>,
    public close: () => Promise<void>,

    /** A bidirectional channel for control messages to the Broker. */
    public messages: MessagePackChannel,

    /** The incoming RPC requests in an `AsyncGenerator`. */
    public rpc: RPCServer,

  ) { };

  // establish the connection
  public static async connect(url: string, certhash?: string): Promise<WebTransportBroker> {

    // assemble options with an optional certificate hash
    let options: WebTransportOptions = { requireUnreliable: true };
    if (certhash !== undefined) {
      options.serverCertificateHashes = [{
        "algorithm": "sha-256",
        "value": Uint8Array.from(certhash.match(/../g)!.map(b => parseInt(b, 16))), // parse hex to bytes
      }];
    };

    // establish connection and wait for readiness
    let transport = new WebTransport(url, options);
    await transport.ready;

    // connect the closed promise from transport
    let closed = transport.closed;
    let close = async () => {
      // TODO: probably needs promise cancellation in async generators to work correctly
      // https://seg.phault.net/blog/2018/03/async-iterators-cancellation/
      // await Promise.allSettled([
      //   (await this.rpc).return(),
      //   (await this.messages).close(),
      // ]);
      return transport.close();
    };

    // open a bidirectional stream for control messages
    let messages = new MessagePackChannel(await transport.createBidirectionalStream());

    // await an incoming bidirectional stream for rpc requests
    let rpc = NetRPCStreamServer(await next(transport.incomingBidirectionalStreams));

    // listen for closure and properly exit the streams
    closed
      .then(async () => Promise.allSettled([ messages.close(), rpc.return() ]))
      .catch(() => { /* don't care */ });

    return new WebTransportBroker(transport, closed, close, messages, rpc);
  };

}

//? +---------------------------------------------------------------+
//? | Wrap a bidirectional WebTransport stream in a net/rpc server. |
//? +---------------------------------------------------------------+


/** Decode MessagePack messages from a `ReadableStream` and yield the decoded RPC requests. */
export async function* NetRPCStreamDecoder(stream: ReadableStream<Uint8Array>): NetRPCDecoder {

  // decode MessagePack encoded messages and yield [ header, body ] pairs for inner loop
  const messages = MessagePack.decodeMultiStream(stream, { useBigInt64: true, context: null });

  try {
    for await (const { 0: header, 1: body } of pairs<NetRPCRequestHeader, any>(messages)) {
      // deconstruct the header and yield request information
      let { ServiceMethod: method, Seq: seq } = header;
      yield { method, seq, body };
    };
  } finally {
    // release the lock on .return()
    messages.return();
  };

};


/** Lock a `WritableStream` and return a function which encodes RPC responses on it. */
export function NetRPCStreamEncoder(stream: WritableStream<Uint8Array>): NetRPCEncoder {

  // get a lock on the writer and create a persistent MessagePack encoder
  const writer = stream.getWriter();
  const msgpack = new MessagePack.Encoder({ useBigInt64: true, initialBufferSize: 65536 });

  return { // anonymous { next, close }

    // encode a chunk as response
    async next(r) {
      // encode the response halves into a single buffer
      // TODO: optimize with less buffer copy operations, e.g. use .encodeSharedRef() or Uint8ArrayList
      let header = msgpack.encode({ ServiceMethod: r.method, Seq: r.seq, Error: r.error } as NetRPCResponseHeader);
      let body = msgpack.encode(r.body as any);
      let buf = new Uint8Array(header.byteLength + body.byteLength);
      buf.set(header, 0); buf.set(body, header.byteLength);
      // wait for possible backpressure on stream and then write
      await writer.ready;
      return writer.write(buf);
    },

    // close the writer
    async close() {
      return writer.close().then(() => writer.releaseLock());
    },

  };
};


/** Wrap a bidirectional WebTransport stream and return an asynchronous generator of RPC requests to handle. */
export async function* NetRPCStreamServer(stream: WebTransportBidirectionalStream): RPCServer {

  // pretty logging prefixes
  const prefixRx   = [ "%c RPC %c « Call %c %s ", "background: #333; color: white;", "background: skyblue;", "background: #ccc;" ];
  const prefixTx   = [ "%c RPC %c Done » %c %s ", "background: #333; color: white;", "background: greenyellow;", "background: #ccc;" ];
  const prefixErr  = [ "%c RPC %c Error ", "background: #333; color: white;", "background: firebrick; color: white;" ];
  const prefixWarn = [ "%c RPC %c Warning ", "background: #333; color: white;", "background: goldenrod;" ];

  // create the net/rpc messsagepack codec on the stream
  const decoder = NetRPCStreamDecoder(stream.readable);
  const encoder = NetRPCStreamEncoder(stream.writable);

  try { // generator loop

    // for each request .. yield a function that must be called with an async handler
    for await (const { method, seq, body } of decoder) {
      console.debug(...prefixRx, method, seq, body);
      yield async (handler) => {
        try {

          // happy path: return result to client
          let result = await handler(method, body);
          console.debug(...prefixTx, method, seq, result);
          await encoder.next({ method, seq, body: result });

        } catch (error) {

          // catch errors and report back to client
          console.warn(...prefixErr, method, seq, error);
          await encoder.next({ method, seq, body: undefined, error: String(error) });

        };
      };
    };
    console.warn(...prefixWarn, "NetRPCStreamDecoder has ended!");

  } finally { // handle .return() by closing streams

    // close both directional streams in parallel
    await Promise.allSettled([

      // release lock on the decoder, then close the readable
      await decoder.return().then(() => stream.readable.cancel()),

      // close and release the writer in the encoder
      //! this will cut off any in-flight requests, unfortunately
      encoder.close(),

    ]);

  };
};


