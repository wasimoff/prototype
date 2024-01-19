# wasimoff webprovider

This is the browser-based provider for the `wasimoff` project.
It was created as a Vue 3 template using `yarn create vue`.

| Action                                | Command      |
| ------------------------------------- | ------------ |
| Install project dependencies          | `yarn`       |
| Serve with hot-reload for development | `yarn dev`   |
| Compile and minify for production     | `yarn build` |

### Recommended IDE Setup

* [VSCode](https://code.visualstudio.com/) with extensions:
  * [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur)
  * [TypeScript Vue Plugin (Volar)](https://marketplace.visualstudio.com/items?itemName=Vue.vscode-typescript-vue-plugin)
* [yarn](https://yarnpkg.com/) package manager

### Required Browser Version

While developing locally, the provider runs better in Google Chrome, due to
[WebTransport bugs in Firefox](https://github.com/quic-go/webtransport-go/issues/84).
When the frontend is deployed somewhere with non-negligible latency, it works
fine in Firefox and it even seems to have a little better WebAssembly performance.

* During development with `yarn dev`, the Workers need to [support loading code from ES modules](https://caniuse.com/mdn-api_worker_worker_ecmascript_modules). When the project is built for production, the imports are bundled into the file.
* The Browser [needs to have WebTransport](https://caniuse.com/webtransport). Previously, I enabled the flag `network.webtransport.enabled` in `about:config` on Firefox because the feature was present but marked as experimental. Since 114 it is fully activated.

#### Running Headless

You can run Chrome headless in a terminal:

```
chromium --headless=new "http://localhost:5173/#autoconnect=yes&workers=max"
```
