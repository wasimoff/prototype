# wasimoff Client

We started work on a "client" implementation for Wasimoff in Go but the
API is so simple that you can do all of that with `curl` or `httpie` and
some JSON crafting.

An **experimental** client written in Bash can be found in `./client.sh`.
It worked great for our purposes but is considered experimental because the
JSON is constructed using `printf`, which is not really safe.

### Set the `BROKER` URL

The script expects the URL to the Broker in the environment variable `BROKER`.
By default it assumes `http://localhost:4080` for a locally deployed Broker on
the same machine. If your Broker is somewhere else, it's best to set this URL
once before running the other commands:

```
export BROKER=https://broker.example.com
```

### Upload a WebAssembly Executable or auxilliary file

WebAssembly executables and data files can be uploaded to the Broker, which
distributes it among Providers.

```
# upload an executable
./client.sh upload mytask.wasm
HTTP/1.1 200 OK
[...]

Upload OK, 188420 bytes
```

```
# upload data file (and store it with another name)
./client.sh upload data.bin [mydata.bin]
[...]
```

### Execute a WebAssembly file

If at least one Provider is connected to the Broker and you've uploaded your
executable, you can run it as if you were running the application locally.
The script currently only forwards commandline arguments. If you need more
customization, check the next section.

```
./client exec mytask.wasm --my task --arguments
POST /api/broker/v1/run HTTP/1.1
[... copy of offloading payload]

HTTP/1.1 200 OK
[...]

[{ "result": { "status": 0, "stderr": "", "stdout": "..." } }]
```

### Run a prepared payload

For more customization prepare a JSON file with an array of tasks. You can take
the payload from a `./client.sh exec` call as a starter. The following is a
configuration to transcode 10 seconds of a movie clip with
[ffmpeg.wasm](https://github.com/SebastiaanYN/FFmpeg-WASI).
It uses `loadfs` to prepare the virtual filesystem with a file from storage
and `datafile` to return a resulting file from the virtual filesystem. Note
that you can configure an array of `exec` objects, which must all use the same
executable.

```json
{
  "bin": "ffmpeg.wasm",
  "exec": [{
    "envs": [ ],
    "args": [ "-i", "/movie.mp4", "-c:v", "h264", "-ss", "00:00", "-to", "00:10", "/clip.mp4" ],
    "loadfs": [ "movie.mp4" ],
    "datafile": "clip.mp4"
  }]
}
```

```
./client.sh upload ffmpeg.wasm
./client.sh upload movie.mp4
./client.sh run payload.json
HTTP/1.1 200 OK
[...]

[{
  "result": {
    "status": 0,
    "stderr": "[ ffmpeg logs ]",
    "stdout": "",
    "datafile": "[ base64-encoded file contents ]"
  }
}]
```

See `examples/` for more examples and `../wasi-apps/` for the necessary
WASI applications.
