# WASI Hello-World

I decided to use WASI as the "system interface" for my WebAssembly tasks, as I wanted to avoid writing my own interface (where I would need to create tooling support in all source languages) and because even the current `wasi_snapshot_preview1` interface is well enough for my tasks:

* it has a commandline with arguments
* it has environment variables
* it has relatively easily useable standard input, output and error
* and it can even provide a filesystem

There's lots more extensions in the works but this is the minimal useful set for me. Targeting WASI also allows me to easily use languages like Rust and Go (TinyGo), which include `wasm32-wasi` as a build target, *without* resorting to things like `#![no_std]` programming.

### tl;dr:

* `rust/` contains a small **Rust** binary, which prints the **commandline arguments**, lists the **root filesystem (`/`)**, if available, and **prints the contents of a specific file** in the filesystem (`hello.txt`)
* `tinygo/` is a **[TinyGo](https://tinygo.org/)** ("a Go compiler for small places") reimplementation of the above
* `wat/` is not a reference to [Gary Bernhardt's talk](https://www.destroyallsoftware.com/talks/wat) but a tiny "WebAssembly Text" hello-world. it only prints this string but also needs no compilation; just a simple translation using `wat2wasm`
