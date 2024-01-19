package provider

import (
	"context"
	"fmt"
	"log"
	"wasmoff/broker/storage"
	"wasmoff/broker/tracer"
)

// --------------- functions for the RPC client ---------------
// the client itself is provided by net/rpc using wasmoff/broker/msgprpc codec

// ----- Misc. -----

// Ping sends a simple "ping" and expects a "pong" back
func (p *Provider) Ping() error {
	var reply string
	err := p.rpc.Call("ping", "ping", &reply)
	if err != nil {
		return fmt.Errorf("rpc call failed: %w", err)
	}
	if reply != "pong" {
		return fmt.Errorf("received wrong reply: %s", reply)
	}
	return nil
}

// ----- WASM -----

// run is the internal detail, which calls the RPC on the Provider
func (p *Provider) run(run *WasmRequest, result *WasmResponse) (err error) {
	log.Printf("RUN %s on %q", run.Id, p.Addr)
	err = p.rpc.Call("run", run, result)
	log.Printf("RESULT %s from %q: %#v", run.Id, p.Addr, result)
	return
}

// Run will run a task on a Provider synchronously
func (p *Provider) Run(run *WasmRequest) (result WasmResponse, err error) {
	p.limiter.Acquire(context.TODO(), 1)
	defer p.limiter.Release(1)
	err = p.run(run, &result)
	return
}

// TryRun will attempt to run a task on the Provider but fails when there is no capacity
func (p *Provider) TryRun(run *WasmRequest) (result WasmResponse, err error) {
	if ok := p.limiter.TryAcquire(1); !ok {
		err = fmt.Errorf("no free capacity")
		return
	}
	defer p.limiter.Release(1)
	err = p.run(run, &result)
	return
}

type WasmRequest struct {
	Id       string   `msgpack:"id"`
	Binary   string   `msgpack:"binary"`
	Args     []string `msgpack:"args"`
	Envs     []string `msgpack:"envs"`
	Stdin    string   `msgpack:"stdin,omitempty"`
	Loadfs   []string `msgpack:"loadfs,omitempty"`
	Datafile string   `msgpack:"datafile,omitempty"`
	Trace    bool     `msgpack:"trace,omitempty"`
}

type WasmResponse struct {
	Status   int            `msgpack:"exitcode"`
	Stdout   string         `msgpack:"stdout"`
	Stderr   string         `msgpack:"stderr"`
	Datafile []byte         `msgpack:"datafile"`
	Trace    []tracer.Event `msgpack:"trace"`
}

// ----- Filesystem -----

// ListFiles will ask the Provider to list their files in storage
func (p *Provider) ListFiles() error {
	// fetch list of files
	var hasfiles []*storage.File
	err := p.rpc.Call("fs:list", nil, &hasfiles)
	if err != nil {
		return fmt.Errorf("rpc call failed: %w", err)
	}
	// clear provider info and set known files to received list
	for k := range p.files {
		delete(p.files, k)
	}
	for _, file := range hasfiles {
		p.files[file.Name] = file
	}
	return nil
}

// ProbeFile sends a name and hash to check if the Provider *has* a file
func (p *Provider) ProbeFile(file *storage.File) (has bool, err error) {
	err = p.rpc.Call("fs:probe", file.CloneWithoutBytes(), &has)
	return
}

// Upload uploads a file to this Provider
func (p *Provider) Upload(file *storage.File) (err error) {
	defer func() {
		// when func returns without an error, add the file to provider's map
		if err == nil {
			p.files[file.Name] = file
			fmt.Printf("Added file to Provider[%v]: %#v\n", p.Addr, p.files)
		}
	}()
	// always probe for file first
	has, err := p.ProbeFile(file)
	if err != nil {
		return fmt.Errorf("failed to probe for existence of file: %w", err)
	}
	if has {
		return // provider has this exact file already
	}
	var ok bool
	err = p.rpc.Call("fs:upload", file, &ok)
	if err == nil && !ok {
		err = fmt.Errorf("upload request returned 'false'")
	}
	return
}

// Has returns if this Provider *is known* to have a certain file, without re-probing
func (p *Provider) Has(file string) bool {
	_, ok := p.files[file]
	return ok
}
