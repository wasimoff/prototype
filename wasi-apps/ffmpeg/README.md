# FFmpeg-WASI

This example uses an FFmpeg binary compiled for the WebAssembly System Interface
(WASI). It is standalone and requires no JavaScript glue; only a suitable WebAssembly
runtime with WASI support.

    wasmtime ./ffmpeg.wasm -- -version

Sources and build instructions can be found at
[github.com/SebastiaanYN/FFmpeg-WASI](https://github.com/SebastiaanYN/FFmpeg-WASI).
It is built in a Docker container, which unfortunately takes a long time. To
avoid licensing issues, the binary is not committed herein though.

## Test Files

For royalty-free video files, you can use the
[Big Buck Bunny Blender movie](https://peach.blender.org/download/).
It has been released by the Blender Foundation in various formats under
a Creative Commons Attribution 3.0 license.

Direct download links can be found here: https://download.blender.org/demo/movies/BBB/

If the above server is overloaded, you can also find the film on
[YouTube](https://www.youtube.com/watch?v=aqz-KE-bpKQ).
