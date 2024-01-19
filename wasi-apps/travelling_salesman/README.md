# Travelling Saleman Problem Workload

This is a simple Rust binary, which solves the [Travelling Salesman Problem (TSP)](https://en.wikipedia.org/wiki/Travelling_salesman_problem) using the deterministic brute-force algorithm from the [`travelling_salesman` crate](https://docs.rs/travelling_salesman/latest/travelling_salesman/).

There's a selection of two datasets in the `src/datasets.rs` file: `WG59`, 50 cities in West Germany, and [`SGB128`, a selection of 128 North American cities](https://people.sc.fsu.edu/~jburkardt/datasets/cities/cities.html).

## Usage

The included `Makefile` can quickly compile binaries for your native architecture or a WASI-compatible WebAssembly binary:

```
make tsp
time ./tsp
```

```
make tsp.wasm
time wasmtime ./tsp.wasm
```

The binary takes one of three arguments:

* `tsp rand [n]` – pick `n` coordinates at random and run the solver
* `tsp write [n]` – pick `n` coordinates at random and serialize as CSV
* `tsp read` – read a CSV from stdin and run the solver on those coordinates
