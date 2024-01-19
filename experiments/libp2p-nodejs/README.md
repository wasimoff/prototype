# Local `libp2p` Example

This is a demonstration using [`js-libp2p`](https://github.com/libp2p/js-libp2p), adapted from [the `auto-relay/` example](https://github.com/libp2p/js-libp2p/tree/master/examples/auto-relay).

* It Uses TCP and CircuitRelay transports.
* The Relay advertises itself for reservations.
* The Listener takes a reservation and starts listening on the relayed transport.
* The Dialer connects to the Relay and participates in the PubSub PeerDiscovery to get the Listener's address.
* In this simple example, you can also directly construct the Listener's address – assuming that it already has a reservation – but this way the demonstration also utilizes a builtin discovery mechanism.

### How to run the demonstration:

1. Run the **Relay** and copy its listening address:
   ```
   $ node run-relay.js
   ...
   [NODE] 12D3KooWD91XkY...
   [LISTEN] [
     '/dns4/localhost/tcp/30000/p2p/12D3KooWD91XkY...'
   ]
   
   ```

2. Start the **Listener** by specifying the address from above as an argument:
   ```
   $ node run-listener.js /dns4/localhost/tcp/30000/p2p/12D3KooWD91XkY...
   ...
   ```

   This will yield a relayed listening address (which begins with the Relay's multiaddress) after a while:
   ```
   ...
   [LISTEN] [
     '/dns4/localhost/tcp/30000/p2p/12D3KooWD91XkY.../p2p-circuit/p2p/12D3KooWAjDkhp...'
   ]
   
   ```

3. Start the **Dialer** to open the chat by specifying the Relay's address and the Listener's Peer ID as arguments:
   ```
   $ node run-dialer.js /dns4/localhost/tcp/30000/p2p/12D3KooWD91XkY... /p2p/12D3KooWAjDkhp...
   ...
   ```

   This will take a moment as the node waits for the desired Peer ID to appear in its PeerStore through PubSub discovery. Then the protocol is opened and you can start chatting between Listener and Dialer.
