// auto-relay server from https://github.com/libp2p/js-libp2p/tree/master/examples/auto-relay

import { createLibp2p } from "libp2p";
import { tcp } from "@libp2p/tcp";
import { noise } from "@chainsafe/libp2p-noise";
import { mplex } from "@libp2p/mplex";
import { circuitRelayServer, circuitRelayTransport } from "libp2p/circuit-relay";
import { identifyService } from "libp2p/identify";
import { createFromJSON } from "@libp2p/peer-id-factory";
import { floodsub as pubsub } from "@libp2p/floodsub";
// import { gossipsub as pubsub } from "@chainsafe/libp2p-gossipsub";
import figlet from "figlet";
import { fromString } from "uint8arrays/from-string";
import { pubsubPeerDiscovery } from "@libp2p/pubsub-peer-discovery";
import * as cm from "./common.js";

// --- prelude -------------------------------------------------------------- //

//! load a persistent peer-id
//? (await require('peer-id').create({ keyType: "Ed25519" })).toJSON()
const peerId = await createFromJSON({
  id: '12D3KooWD91XkY9wXXwQBXYWoLdS5EiB3fu3MXoax2X3erowywwK',
  privKey: 'CAESQKLqzq10v248o2IILSAKRdcNcY6Vm3poGyVs6tFl3c5eMVnCyqzS8GNQl3+cpKRD5CaxlY4o1smr+AQSbYrHv4Y=',
  pubKey: 'CAESIDFZwsqs0vBjUJd/nKSkQ+QmsZWOKNbJq/gEEm2Kx7+G'
});

// print banner
console.log(figlet.textSync("auto relay", { font: "Small" }));

// --- constructor ---------------------------------------------------------- //

// create the server node
const node = await createLibp2p({
  peerId,

  // use simple tcp for now
  transports: [ tcp(), circuitRelayTransport() ],
  connectionEncryption: [ noise() ],
  streamMuxers: [ mplex() ],

  // how are we reachable
  addresses: {
    listen: [
      "/ip4/127.0.0.1/tcp/30000"
      // "/ip4/127.0.0.1/udp/30000/quic-v1/webtransport", //! not supported in js
    ],
    // announce different, publicly reachable addresses
    announce: [
      "/dns4/localhost/tcp/30000/p2p/12D3KooWD91XkY9wXXwQBXYWoLdS5EiB3fu3MXoax2X3erowywwK"
    ],
  },

  peerDiscovery: [
    pubsubPeerDiscovery({
      interval: 1000,
      topics: [ "wasimoff/discovery" ],
    }),
  ],

  // register services for a relay
  services: {
    identify: identifyService(),
    relay: circuitRelayServer({
      advertise: {},
      reservations: {
        maxReservations: Infinity,
      },
    }),
    pubsub: pubsub(),
  },

});

// --- implementation ------------------------------------------------------- //

// log information
cm.printNodeId(node);
cm.printListeningAddrs(node);
cm.printPeerConnections(node);
cm.printPeerDiscoveries(node);
cm.printPeerStoreUpdates(node, "peer:update");
