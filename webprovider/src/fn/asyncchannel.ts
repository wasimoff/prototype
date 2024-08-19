import * as MessagePack from "@msgpack/msgpack";

// a generic wrapper which makes bidirectional streams easier to use
export interface AsyncChannel<Value> {
  channel: AsyncGenerator<Value, void, undefined>;
  send: (value: Value) => Promise<void>;
  close: () => Promise<void>;
};

// an implementation of the above, that was previously used on a webtransport stream
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