package storage

import (
	"fmt"
	"log"
	"wasimoff/broker/net/pb"
)

// TODO: use SQLite, BoltDB or just filesystem for persistence
// TODO: this storage is not threadsafe

type FileStorage struct {
	// collection of Files in storage, keyed by content address
	Files map[string]*File
	// a Lookup table of plain names to content addresses
	Lookup map[string]string
}

func NewFileStorage() FileStorage {
	return FileStorage{
		Files:  make(map[string]*File),
		Lookup: make(map[string]string),
	}
}

// Insert a new file into the Storage. The optional `name` will be inserted
// into the lookup table and can be used to resolve the file later.
func (fs *FileStorage) Insert(name, media string, blob []byte) (file *File, err error) {

	// check the media type first because that's cheapest
	media, err = CheckMediaType(media)
	if err != nil {
		return nil, fmt.Errorf("media: %w", err)
	}

	// we could check if the file exists already here but since we operate on
	// memory for now, we can just overwrite whatever is there cheaply
	file = NewFile(media, blob)
	ref := file.Ref()
	fs.Files[ref] = file

	// maybe insert name in lookup map, if given
	if name != "" {
		fs.Lookup[name] = ref
	}
	fs.debug()
	return
}

// Get a File from Storage, either by Ref or a friendly name in lookup map.
func (fs *FileStorage) Get(nameOrRef string) *File {
	// try from files directly first
	if file := fs.Files[nameOrRef]; file != nil {
		return file
	}
	// or lookup a friendly name
	if ref, ok := fs.Lookup[nameOrRef]; ok {
		return fs.Files[ref]
	}
	return nil
}

// ResolvePbFile checks if this file is usable as an argument in offloading
// requests, i.e. if it either contains a blob or is a known file in the
// storage. If so, set the resolved Ref on the file.
func (fs *FileStorage) ResolvePbFile(pbf *pb.File) error {

	// argument is nil, no need to do anything
	if pbf == nil {
		return nil
	}

	// trivial errors when both are nil or both are given
	if pbf.Blob == nil && pbf.Ref == nil {
		return fmt.Errorf("both Blob and Ref are nil")
	}
	if pbf.Blob != nil && pbf.Ref != nil {
		return fmt.Errorf("don't use both Blob and Ref together")
	}

	// Blob is given directly, ok ...
	if pbf.Blob != nil {
		// check the media type, if given
		if mt := pbf.GetMedia(); mt != "" {
			mt, err := CheckMediaType(mt)
			if err != nil {
				return fmt.Errorf("invalid Media type")
			}
			pbf.Media = &mt
		}
		return nil
	}

	// Ref is given, look it up in Storage
	if file := fs.Get(*pbf.Ref); file != nil {
		pbf.Media = &file.Media
		pbf.Ref = &file.ref
		return nil
	}

	// couldn't resolve the file
	return fmt.Errorf("Ref not found in storage")

}

func (fs *FileStorage) debug() {
	log.Println("Inserted in FileStorage:")
	for k, v := range fs.Lookup {
		fmt.Printf(" %s => %s\n", k, v)
	}
	for k, f := range fs.Files {
		fmt.Printf("+ %s => %s, %d bytes\n", k, f.Media, len(f.Bytes))
	}
}
