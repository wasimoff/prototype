package provider

import (
	"context"
	"fmt"
	"log"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/storage"
)

// TODO: make this a main.go config, via WASIMOFF_DEBUG as well?
const debuglog = false

// TODO: rewrite using new log/slog package and slog.Debug()
func printdbg(format string, v ...any) {
	if debuglog {
		log.Printf(format, v...)
	}
}

// ----- execute -----

// run is the internal detail, which executes a task on the Provider without semaphore guards
func (p *Provider) run(ctx context.Context, args *pb.Task_Request, result *pb.Task_Response) (err error) {
	addr := p.Get(Address)
	task := args.GetInfo().GetId()
	printdbg("scheduled >> %s >> %s", task, addr)
	if err := p.messenger.RequestSync(ctx, args, result); err != nil {
		printdbg("ERROR!    << %s << %s", task, addr)
		return fmt.Errorf("provider.run failed: %w", err)
	}
	// TODO: add a safeguard that result contains correct type matching args?
	printdbg("finished  << %s << %s", task, addr)
	return
}

// Run will run a task on a Provider synchronously, respecting limiter.
func (p *Provider) Run(ctx context.Context, args *pb.Task_Request, result *pb.Task_Response) error {
	p.limiter.Acquire(context.TODO(), 1)
	defer p.limiter.Release(1)
	return p.run(ctx, args, result)
}

// TryRun will attempt to run a task on the Provider but fails when there is no capacity.
func (p *Provider) TryRun(ctx context.Context, args *pb.Task_Request, result *pb.Task_Response) error {
	if ok := p.limiter.TryAcquire(1); !ok {
		return fmt.Errorf("no free capacity")
	}
	defer p.limiter.Release(1)
	return p.run(ctx, args, result)
}

// ----- filesystem -----

// ListFiles asks the Provider to list their files in storage
func (p *Provider) ListFiles() (map[string]struct{}, error) {

	// receive listing into a new struct
	args := pb.FileListingRequest{}
	response := pb.FileListingResponse{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return nil, fmt.Errorf("provider.ListFiles failed: %w", err)
	}

	// (re)set known files from received list
	p.files = make(map[string]struct{})
	for _, filename := range response.Files {
		p.files[filename] = struct{}{}
	}

	return p.files, nil
}

// ProbeFile sends a content-address name to check if the Provider *has* a file
func (p *Provider) ProbeFile(addr string) (has bool, err error) {

	// receive response bool into a new struct
	args := pb.FileProbeRequest{File: &addr}
	response := pb.FileProbeResponse{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return false, fmt.Errorf("provider.ProbeFile failed: %w", err)
	}

	return response.GetOk(), nil
}

// Upload a file from Storage to this Provider
func (p *Provider) Upload(file *storage.File) (err error) {
	ref := file.Ref()

	// when returning without error, add the file to provider's list
	// (either probe was ok or upload successful)
	defer func() {
		if err == nil {
			p.files[ref] = struct{}{}
		}
	}()

	// always probe for file first
	if has, err := p.ProbeFile(ref); err != nil {
		return fmt.Errorf("provider.Upload failed probe before upload: %w", err)
	} else if has {
		return nil // ok, provider has this file already
	}

	// otherwise upload it
	args := pb.FileUploadRequest{Upload: &pb.File{
		Ref:   &ref,
		Media: &file.Media,
		Blob:  file.Bytes,
	}}
	response := pb.FileUploadResponse{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return fmt.Errorf("provider.Upload %q failed: %w", ref, err)
	}
	if response.GetErr() != "" {
		return fmt.Errorf("provider.Upload %q failed at Provider: %s", ref, *response.Err)
	}
	return
}

// Has returns if this Provider *is known* to have a certain file, without re-probing
func (p *Provider) Has(file string) bool {
	_, ok := p.files[file]
	return ok
}
