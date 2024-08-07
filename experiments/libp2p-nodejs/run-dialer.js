// auto-relay dialer from https://github.com/libp2p/js-libp2p/tree/master/examples/auto-relay

import { createLibp2p } from "libp2p";
import { tcp } from "@libp2p/tcp";
import { noise } from "@chainsafe/libp2p-noise";
import { mplex } from "@libp2p/mplex";
import { multiaddr } from "@multiformats/multiaddr";
import { circuitRelayTransport } from "libp2p/circuit-relay";
import { identifyService } from "libp2p/identify";
import { createFromJSON } from "@libp2p/peer-id-factory";
import { peerIdFromString } from "@libp2p/peer-id";
import { bootstrap } from "@libp2p/bootstrap";
import { floodsub as pubsub } from "@libp2p/floodsub";
// import { gossipsub as pubsub } from "@chainsafe/libp2p-gossipsub";
import { pubsubPeerDiscovery } from "@libp2p/pubsub-peer-discovery";
import figlet from "figlet";
import { stdinToStream, streamToConsole } from "./stream.js";
import * as cm from "./common.js";

// --- prelude -------------------------------------------------------------- //

//! load a persistent peer-id
//? (await require('peer-id').create({ keyType: "Ed25519" })).toJSON()
const peerId = await createFromJSON({
  id: '12D3KooWQAkbxDYXsRBL75hzjYXDWq2ntT7U8zA2DLsVPRnEtRTg',
  privKey: 'CAESQF+mPW4UsUKHjDjAGAqC4WLVFKOlQAEQx5cQxurgM2Nf1TyZnc4eFI3TJLmQSar77bQaWrICjvIm1N14StIzd1M=',
  pubKey: 'CAESINU8mZ3OHhSN0yS5kEmq++20GlqyAo7yJtTdeErSM3dT'
});

// print banner
console.log(figlet.textSync("auto dialer", { font: "Small" }));

// relay address expected in argument
if (!process.argv[2]) throw new Error("relay address expected in first argument");
const relay = multiaddr(process.argv[2]);

// peer id expected in argument
if (!process.argv[3]) throw new Error("peer id expected in second argument");
const peer = multiaddr(process.argv[3]);

// --- constructor ---------------------------------------------------------- //

// create the client node
const node = await createLibp2p({
  peerId,

  // use simple tcp for now but append relay transport
  transports: [
    tcp(),
    circuitRelayTransport(),
  ],
  connectionEncryption: [ noise() ],
  streamMuxers: [ mplex() ],

  // we don't need to advertise anything
  // addresses: { ... }

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
cm.printPeerStoreUpdates(node, "peer:identify");


// which dial to use
const waitForDiscovery = true;

function chatStream(stream) {
  console.log("--- stream opened ---");
  stdinToStream(stream);
  streamToConsole(stream);
};

(async () => {
  if (waitForDiscovery) {

    // wait until peerid is routable
    const pid = peerIdFromString(peer.getPeerId());
    console.log("wait for peerId:", peer);
    while (!await node.peerStore.has(pid) || !(await node.peerStore.get(pid)).addresses.length) {
      await new Promise(r => setTimeout(r, 100));
    };

    // try to dial the requested peer for chat
    chatStream(await node.dialProtocol(pid, "/wasimoff/chat/v1"));

  } else {

    // directly dial the peer by encapsulating in relay address
    const relayed = relay.encapsulate("/p2p-circuit").encapsulate(peer);
    chatStream(await node.dialProtocol(relayed, "/wasimoff/chat/v1"));

  };
})();

// dial the peer through the relay
// const conn = await node.dial(peer);
// console.log("Connected to peer:", conn.remotePeer.toString());

