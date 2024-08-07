// auto-relay listener from https://github.com/libp2p/js-libp2p/tree/master/examples/auto-relay

import { createLibp2p } from "libp2p";
import { tcp } from "@libp2p/tcp";
import { noise } from "@chainsafe/libp2p-noise";
import { mplex } from "@libp2p/mplex";
import { multiaddr } from "@multiformats/multiaddr";
import { circuitRelayTransport,  } from "libp2p/circuit-relay";
import { identifyService } from "libp2p/identify";
import { createFromJSON } from "@libp2p/peer-id-factory";
import { bootstrap } from "@libp2p/bootstrap";
import { floodsub as pubsub } from "@libp2p/floodsub";
// import { gossipsub as pubsub } from "@chainsafe/libp2p-gossipsub";
import { pubsubPeerDiscovery } from "@libp2p/pubsub-peer-discovery";
import figlet from "figlet";
import { iostream, stdinToStream, streamToConsole } from "./stream.js";
import * as cm from "./common.js";

// --- prelude -------------------------------------------------------------- //

//! load a persistent peer-id
//? (await require('peer-id').create({ keyType: "Ed25519" })).toJSON()
const peerId = await createFromJSON({
  id: '12D3KooWAjDkhpNMdFbmd7V4eoMBis2KKgwDwNMuiqXbT4RfCd9Z',
  privKey: 'CAESQGg0DbD4P5WuGEy2BW+rD0bvFOxy2eO8/dr5emmGiFXJDYpEwsXmVoU21XBT5GAQrHgokJWqguHPSiZHDzEG69g=',
  pubKey: 'CAESIA2KRMLF5laFNtVwU+RgEKx4KJCVqoLhz0omRw8xBuvY'
});

// print banner
console.log(figlet.textSync("auto listener", { font: "Small" }));

// relay address expected in argument
if (!process.argv[2]) throw new Error("relay address expected in argument");
const relay = multiaddr(process.argv[2]);

// --- constructor ---------------------------------------------------------- //

// create the client node
const node = await createLibp2p({
  peerId,

  // use simple tcp for now but append relay transport
  transports: [
    tcp(),
    circuitRelayTransport({
      discoverRelays: 1,
    }),
  ],
  connectionEncryption: [ noise() ],
  streamMuxers: [ mplex() ],

  // try to use a reservation on the relay
  addresses: {
    listen: [ relay.encapsulate("/p2p-circuit").toString() ],
  },

  // discover peers through the relay to avoid manually dialing
  peerDiscovery: [
    bootstrap({
      list: [ relay ],
    }),
    pubsubPeerDiscovery({
      interval: 1000,
      topics: [ "wasimoff/discovery" ],
    }),
  ],

  // add services for relaying
  services: {
    identify: identifyService(),
    pubsub: pubsub(),
  },

});

// --- implementation ------------------------------------------------------- //

// log information
cm.printNodeId(node);
cm.printPeerConnections(node);
cm.printPeerDiscoveries(node);
cm.printListeningAddrs(node); // print own multiaddr once we have a relay
cm.printPeerStoreUpdates(node, "peer:update");

// handle a simple chat protocol
await node.handle("/wasimoff/chat/v1", async ({ stream }) => {
  console.log("--- opened chat stream ---");
  // await iostream(stream);
  stdinToStream(stream);
  streamToConsole(stream);
});

// debug the pubsub messages
// import { toString } from "uint8arrays/to-string";
// node.services.pubsub.addEventListener("message", event => {
//   console.log(`  pubsub: ${event.detail.topic}:`, toString(event.detail.data));
// });