package provider

import (
	"context"
	"fmt"
	"log"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/storage"
)

const debuglog = false

func printdbg(format string, v ...any) {
	if debuglog {
		log.Printf(format, v...)
	}
}

// ----- execute -----

// run is the internal detail, which executes a WASI binary on the Provider without semaphore guards
func (p *Provider) run(args *pb.ExecuteWasiArgs, result *pb.ExecuteWasiResult) (err error) {
	addr := p.Get(Address)
	task := fmt.Sprintf("%s/%d", *args.Task.Id, *args.Task.Index)
	printdbg("scheduled >> %s >> %s", task, addr)
	if err := p.messenger.RequestSync(context.TODO(), args, result); err != nil {
		printdbg("ERROR!    << %s << %s", task, addr)
		return fmt.Errorf("provider.run failed: %w", err)
	}
	printdbg("finished  << %s << %s", task, addr)
	return
}

// Run will run a task on a Provider synchronously, respecting limiter.
func (p *Provider) Run(args *pb.ExecuteWasiArgs, result *pb.ExecuteWasiResult) error {
	p.limiter.Acquire(context.TODO(), 1)
	defer p.limiter.Release(1)
	return p.run(args, result)
}

// TryRun will attempt to run a task on the Provider but fails when there is no capacity.
func (p *Provider) TryRun(args *pb.ExecuteWasiArgs, result *pb.ExecuteWasiResult) error {
	if ok := p.limiter.TryAcquire(1); !ok {
		return fmt.Errorf("no free capacity")
	}
	defer p.limiter.Release(1)
	return p.run(args, result)
}

// ----- filesystem -----

// ListFiles asks the Provider to list their files in storage
func (p *Provider) ListFiles() error {
	// receive listing into a new struct
	files := new(pb.FileListingResult)
	if err := p.messenger.RequestSync(context.TODO(), &pb.FileListingArgs{}, files); err != nil {
		return fmt.Errorf("provider.ListFiles failed: %w", err)
	}
	// (re)set known files from received list
	for k := range p.files {
		delete(p.files, k)
	}
	for _, file := range files.Files {
		if file == nil || file.Filename == nil {
			break // oops
		}
		p.files[*file.Filename] = &storage.File{
			Name:   *file.Filename,
			Hash:   [32]byte(file.GetHash()),
			Length: uint64(file.GetLength()),
			Epoch:  int64(file.GetEpoch()),
		}
	}
	return nil
}

// ProbeFile sends a name and hash to check if the Provider *has* a file
func (p *Provider) ProbeFile(file *storage.File) (has bool, err error) {
	// receive response bool into a new struct
	result := new(pb.FileProbeResult)
	if err := p.messenger.RequestSync(context.TODO(),
		&pb.FileProbeArgs{Stat: &pb.FileStat{
			Filename: &file.Name,
			Length:   &file.Length,
			Epoch:    &file.Epoch,
			Hash:     file.Hash[:],
		}}, result); err != nil {
		return false, fmt.Errorf("provider.ProbeFile failed: %w", err)
	}
	return result.GetOk(), nil
}

// Upload a file to this Provider
func (p *Provider) Upload(file *storage.File) (err error) {
	defer func() {
		// when returning without an error, add the file to provider's map
		if err == nil {
			p.files[file.Name] = file
		}
	}()
	// always probe for file first
	if has, err := p.ProbeFile(file); err != nil {
		return fmt.Errorf("provider.Upload failed probe before upload: %w", err)
	} else if has {
		return nil // NOP, provider has this exact file already
	}
	// otherwise upload it
	result := new(pb.FileUploadResult)
	if err := p.messenger.RequestSync(context.TODO(),
		&pb.FileUploadArgs{
			Stat: &pb.FileStat{
				Filename: &file.Name,
				Length:   &file.Length,
				Epoch:    &file.Epoch,
				Hash:     file.Hash[:],
			},
			File: file.Bytes,
		}, result); err != nil {
		return fmt.Errorf("provider.Upload %q failed: %w", file.Name, err)
	}
	if !result.GetOk() {
		return fmt.Errorf("provider.Upload %q failed at Provider", file.Name)
	}
	return
}

// Has returns if this Provider *is known* to have a certain file, without re-probing
func (p *Provider) Has(file string) bool {
	_, ok := p.files[file]
	return ok
}
