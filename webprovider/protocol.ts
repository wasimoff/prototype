// DRAFT: wasimoff protocol
// Please note that this does not actually correspond to the current
// protocol on the wire! It was a draft to see how well TypeScript
// syntax is suited to describe something like this.






// -- ROLES
// There are ~ four roles in the wasimoff network:
enum Role {
  Broker,   // the Go server, scheduler and "central" entity
  Storage,  // storage for binaries and archives, will be on Broker at first
  Provider, // a node "providing" its computational resources, i.e. Vue frontend
  Thread,   // a single Worker thread of a Provider, ~ Runner
  Client,   // a client sending tasks to the network
  This,     // just a signifier to mean whatever the current context is
};

// -- STREAM TYPES
// In WebTransport there's datagrams and unidirectional or bidirectional Streams.
enum Stream { Datagrams, Unidirectional, Bidirectional };

// -- DIRECTION
// Generally Messages go over a Stream and have an initiator and a recipient Role.
type Direction = {
  initiator: Role,
  recipient: Role,
  type: Stream,
};

// -- MESSAGE
// Each Connection will send and receive Messages with a certain form.
type Message = (name: any) => void;

// -- CONNECTION
// Messages are exchanged over a Connection with a Direction and have
// distinct types for sent and received packets.
type Connection = {
  stream: Direction,
  sending: Message[],
  receiving: Message[],
};


// -----------------------

// When a Provider initially connects to the Broker, it tells how many
// threads it is able and willing to run in parallel. In turn, it receives
// a URL to the storage to use and waits to receive new Streams.
const ProviderConnection: Connection = {
  stream: {
    initiator: Role.Provider,
    recipient: Role.Broker,
    type: Stream.Bidirectional,
  },

  sending: [
    // tell how many threads in parallel
    hello => { let nprocs: number },
  ],

  receiving: [
    // the URL to the storage to use
    storage_link => { let storage: URL | 'self' },
    // open a new stream for each thread
    start_threads => { for (let thread of ["nprocs"]) { RunnerConnection } },
  ],

};

// After telling the Broker how many threads you're willing to run, it will
// open as many bidirectional Streams and each one shall be forwarded to a
// new WASM Runner instance.
const RunnerConnection: Connection = {
  stream: {
    initiator: Role.Broker,
    recipient: Role.Thread,
    type: Stream.Bidirectional,
  },

  sending: [
    // "run" requests to execute some WASM
    run => { // RPC request
      let binary: ArrayBuffer | "hash" | URL;
      let args: string[];"..."
      let envs: string[];
      let filesystem: unknown // TODO: stdin? preopened files? opfs filename?
    },
  ],

  receiving: [
    // results of the run requests
    result => { // RPC response
      let stdout: ArrayBuffer; // can be binary
      let filesystem: unknown; // TODO: request to upload specific files?
      let data: JSON; // TODO: implement through "special" file?
    },
  ],

}

// The Broker also tells the Provider where to download binaries and filesystems,
// so the Provider opens a new bidirectional Stream for transfers. This can be the
// Broker itself, but maybe on a different endpoint.
const StorageConnection: Connection = {
  stream: {
    initiator: Role.Provider,
    recipient: Role.Storage,
    type: Stream.Bidirectional,
  },

  sending: [
    // the "run" requests may include references to files that need to be downloaded.
    // this happens asynchronously before starting the instance ..
    file_request => { let file_id: string; },
  ],

  receiving: [
    // the storage node replies with the contents of the file ..
    download => { let file: ArrayBuffer; },
  ],

}
