package storage

import (
	"fmt"
	"iter"
	"wasimoff/broker/net/pb"
)

// TODO: use SQLite, BoltDB or just filesystem for persistence
// TODO: this storage is not threadsafe

type AbstractFileStorage interface {
	Insert(name, media string, blob []byte) (file *File, err error)
	Get(nameOrRef string) *File
	All() iter.Seq2[string, *File]
}

type FileStorage struct {
	AbstractFileStorage
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
