package provider

import (
	"context"
	"fmt"
	"log"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/storage"

	"google.golang.org/protobuf/proto"
)

// --------------- functions for the RPC client ---------------
// the client itself is provided by net/rpc using wasimoff/broker/msgprpc codec

// ----- WASM -----

// run is the internal detail, which calls the RPC on the Provider
func (p *Provider) run(run *pb.ExecuteWasiArgs) (result *pb.ExecuteWasiResult, err error) {
	log.Printf("RUN %s on %q", *run.Task.Id, p.Addr)
	response, err := p.messenger.RequestSync(&pb.Request{Request: &pb.Request_ExecuteWasiArgs{
		ExecuteWasiArgs: run,
	}})
	if err != nil {
		return nil, err
	}
	result = response.GetExecuteWasiResult()
	log.Printf("RESULT %s from %q: %#v", *run.Task.Id, p.Addr, result)
	return
}

// Run will run a task on a Provider synchronously
func (p *Provider) Run(run *pb.ExecuteWasiArgs) (result *pb.ExecuteWasiResult, err error) {
	p.limiter.Acquire(context.TODO(), 1)
	defer p.limiter.Release(1)
	result, err = p.run(run)
	return
}

// TryRun will attempt to run a task on the Provider but fails when there is no capacity
func (p *Provider) TryRun(run *pb.ExecuteWasiArgs) (result *pb.ExecuteWasiResult, err error) {
	if ok := p.limiter.TryAcquire(1); !ok {
		err = fmt.Errorf("no free capacity")
		return
	}
	defer p.limiter.Release(1)
	result, err = p.run(run)
	return
}

// ----- Filesystem -----

// wrap around a RequestSync and check response before handing it back
func (p *Provider) checkedrequest(request *pb.Request) (*pb.Response, error) {
	response, err := p.messenger.RequestSync(request)
	if err != nil {
		return nil, err
	}
	if response.Error != nil && *response.Error != "" {
		return nil, fmt.Errorf("%s", *response.Error)
	}
	return response, nil
}

// ListFiles will ask the Provider to list their files in storage
func (p *Provider) ListFiles() error {
	response, err := p.checkedrequest(&pb.Request{Request: &pb.Request_FileListingArgs{
		FileListingArgs: &pb.FileListingArgs{},
	}})
	if err != nil {
		return fmt.Errorf("FileListing RPC failed: %w", err)
	}
	r := response.GetFileListingResult()
	if r == nil {
		return fmt.Errorf("FileListing RPC got an empty response")
	}
	// (re)set known files from received list
	for k := range p.files {
		delete(p.files, k)
	}
	for _, file := range r.Files {
		p.files[*file.Filename] = &storage.File{
			Name:   *file.Filename,
			Hash:   [32]byte(file.Hash),
			Length: uint64(*file.Length),
			Epoch:  int64(*file.Epoch),
		}
	}
	return nil
}

// ProbeFile sends a name and hash to check if the Provider *has* a file
func (p *Provider) ProbeFile(file *storage.File) (has bool, err error) {
	// TODO: previously used file.CloneWithoutBytes() here, needed?
	response, err := p.checkedrequest(&pb.Request{Request: &pb.Request_FileProbeArgs{
		FileProbeArgs: &pb.FileProbeArgs{
			File: &pb.FileStat{
				Filename: &file.Name,
				Length:   proto.Uint64(file.Length),
				Epoch:    proto.Int64(file.Epoch),
				Hash:     file.Hash[:],
			},
		},
	}})
	if err != nil {
		return false, fmt.Errorf("FileProbe RPC failed: %w", err)
	}
	r := response.GetFileProbeResult()
	if r == nil {
		return false, fmt.Errorf("FileProbe RPC got an empty response")
	}
	return r.GetOk(), nil
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
	// otherwise upload it
	response, err := p.checkedrequest(&pb.Request{Request: &pb.Request_FileUploadArgs{
		FileUploadArgs: &pb.FileUploadArgs{
			Stat: &pb.FileStat{
				Filename: &file.Name,
				Length:   proto.Uint64(file.Length),
				Epoch:    proto.Int64(file.Epoch),
				Hash:     file.Hash[:],
			},
			File: file.Bytes,
		},
	}})
	if err != nil {
		return fmt.Errorf("FileUpload RPC failed: %w", err)
	}
	r := response.GetFileUploadResult()
	if r == nil {
		return fmt.Errorf("FileUpload RPC got an empty response")
	}
	if !r.GetOk() {
		return fmt.Errorf("FileUpload was not successful")
	}
	return
}

// Has returns if this Provider *is known* to have a certain file, without re-probing
func (p *Provider) Has(file string) bool {
	_, ok := p.files[file]
	return ok
}
