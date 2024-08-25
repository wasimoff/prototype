# denoprovider

This is a quick'n'dirty MVP of a Provider in a Terminal using [Deno](https://docs.deno.com/). There's lots of commented-out stuff, storage is in-memory, connection errors just crash the app etc. It was created by mostly copying the core of the current implementation from `../webprovider/` and adapting what was needed to run in Deno.

* Install [Deno](https://docs.deno.com/).
* `deno run --allow-env --allow-read=./ --allow-net main.ts [args]`
  * optional: `--workers n`: specify the number of Workers to use (one per logical processor by default)
  * optional: `--url ws://...`: URL to the Broker's WebSocket path (`ws://localhost:4080/websocket/provider` for a locally-running Broker by default)