# WebTransport Example

[The WebTransport document](https://www.w3.org/TR/2023/WD-webtransport-20230405/) is still a draft and is not standardized. Luckily, it has already been implemented in the major browsers. Firefox merely requires you to set the flag `network.webtransport.enabled` to `true`. It should be enable in Chrome without modifications.

**However**, both browsers implement the standards a little differently – especially with regards to TLS certifiate security. While Firefox trusts the certificates in its CA store and enables you to use self-signed certificates for WebTransport as long as the signing CA is added to your trusted authorities, Chrome only allows self-signed certificates that are very short-lived use elliptic curves. When you deploy the server with a "proper" certificate signed by a publicly-trusted authority, both work fine.

### Server

First, make sure you have valid certificates in `localhost.{crt,key}`. Either get signed certificates from elsewhere (or via `mkcert`) or use `gencerts.sh` to generate a Chrome-compatible pair. The certificate hash that is printed at the end needs to be updated in `index.html`.

To start the `quic-go/webtransport-go` server, simply run `go run server.go`:

```
$ go run server.go
2023/05/24 17:07:05 static files listening on https://localhost:4443/
2023/05/24 17:07:05 webtransport listening on https://localhost:4443/transport (UDP)
2023/05/24 17:13:51 WebTransport connection attempt
2023/05/24 17:13:51 accepting streams from 127.0.0.1:43347
2023/05/24 17:13:51 running echo handler for stream
2023/05/24 17:13:52 successfully copied stream
```

Now you can either open the page in a browser or connect with another compatible client. Note that there are two listeners.

The server implements a very simple handler that only accepts bidirectional streams and echoes back any bytes it receives.

### Browser

First, see the compatability notes above. You should be able to open the page `https://localhost:4443/` in a browser and see some output in a `<ul>` list. The script will attempt to open a WebTransport connection, open a bidirectional stream and write two strings. Any received buffers are converted back to strings and appended to the list. Afterwards, the connection is gracefully closed.

The output should look similar to this run:

```
Wed, 24 May 2023 14:46:11 GMT | Establish a connection to https://localhost:4443/transport
Wed, 24 May 2023 14:46:11 GMT | WebTransport is ready!
Wed, 24 May 2023 14:46:11 GMT | Opened bidirectional stream.
Wed, 24 May 2023 14:46:11 GMT | WRITE "Hello, WebTransport!"
Wed, 24 May 2023 14:46:11 GMT | WRITE "This is a test."
Wed, 24 May 2023 14:46:11 GMT | READ "Hello, WebTransport!"
Wed, 24 May 2023 14:46:11 GMT | READ "This is a test."
Wed, 24 May 2023 14:46:12 GMT | Closed stream writer.
Wed, 24 May 2023 14:46:12 GMT | Reader is finished.
Wed, 24 May 2023 14:46:14 GMT | Closing connection.
Wed, 24 May 2023 14:46:14 GMT | WebTransport closed gracefully
```

Occasionally it may happen that the browser is so quick to write the strings, that the server returns them both in a single chunk. That means that any protocols that use this stream **need to use a wire format that includes length-encoding**.

```
Wed, 24 May 2023 15:13:56 GMT | READ "Hello, WebTransport!This is a test."
```

#### webtransport.day

As an added bonus, the simple echo handler also works with the public WebTransport demo at [webtransport.day](https://webtransport.day/). Simply enter the localhost transport URL into the field and click "Connect". Then enter some text and click "Send Data". Obviously, only the bidirectional stream will echo anything back.

```
URL: https://localhost:4443/transport
*Connect*

Textfield: bla bla bla
*Open a bidirectional stream*
*Send data*

Event Log:
    Initiating connection...
    Connection ready.
    Datagram writer ready.
    Datagram reader ready.
    Opened bidirectional stream #1 with data: bla bla bla
    Received data on stream #1: bla bla bla
    Stream #1 closed
```

### Go Client

There is also a small Go client, that connects to the above server and performs the same echo test. Run it with `go run ./client`:

```
$ go run ./client
2023/05/24 17:17:29 dial https://localhost:4443/transport
2023/05/24 17:17:29 response: &{200 OK 200 HTTP/3.0 3 0 map[Sec-Webtransport-Http3-Draft:[draft02]] 0xc00032a040 0 [] false false map[] 0xc0000f4100 0xc0000ce370}
2023/05/24 17:17:29 open stream
2023/05/24 17:17:29 write a few bytes to stream
2023/05/24 17:17:29 read bytes from stream
2023/05/24 17:17:29 echoed back 13 bytes: "Hello, World!"

```

