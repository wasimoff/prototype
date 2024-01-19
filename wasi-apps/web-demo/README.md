# Minimal WASI Demo

A minimal `index.html`, which loads a WASI shim for the browser and then
executes one of the "helloworld" binaries ✨*inside the browser*✨.

Go to `../helloworld/` and compile one of the binaries (`rust`, `tinygo` or `wat`)
and symlink the `exe.wasm` here. Then serve the directory contents and open in a
browser, e.g. with `python -m http.server`.
