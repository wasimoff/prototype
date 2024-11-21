package storage

import (
	"fmt"
	"iter"
	"log"
)

// TODO: this storage is probably not threadsafe

type MemoryFileStorage struct {
	// collection of files in storage, keyed by content address
	files map[string]*File
	// a lookup table of plain names to content addresses
	lookup map[string]string
}

func NewMemoryFileStorage() *FileStorage {
	return &FileStorage{
		AbstractFileStorage: &MemoryFileStorage{
			files:  make(map[string]*File),
			lookup: make(map[string]string),
		},
	}
}

// Insert a new file into the Storage. The optional `name` will be inserted
// into the lookup table and can be used to resolve the file later.
func (fs *MemoryFileStorage) Insert(name, media string, blob []byte) (file *File, err error) {

	// check the media type first because that's cheapest
	media, err = CheckMediaType(media)
	if err != nil {
		return nil, fmt.Errorf("media: %w", err)
	}

	// we could check if the file exists already here but since we operate on
	// memory for now, we can just overwrite whatever is there cheaply
	file = NewFile(media, blob)
	ref := file.Ref()
	fs.files[ref] = file

	// maybe insert name in lookup map, if given
	if name != "" {
		fs.lookup[name] = ref
	}
	fs.debug()
	return
}

// Get a File from Storage, either by Ref or a friendly name in lookup map.
func (fs *MemoryFileStorage) Get(nameOrRef string) *File {
	// try from files directly first
	if file := fs.files[nameOrRef]; file != nil {
		return file
	}
	// or lookup a friendly name
	if ref, ok := fs.lookup[nameOrRef]; ok {
		return fs.files[ref]
	}
	return nil
}

// Iterator over all Files in the storage.
func (fs *MemoryFileStorage) All() iter.Seq2[string, *File] {
	return func(yield func(string, *File) bool) {
		for ref, file := range fs.files {
			if !yield(ref, file) {
				return
			}
		}
	}
}

func (fs *MemoryFileStorage) debug() {
	log.Println("Inserted in MemoryFileStorage:")
	for k, v := range fs.lookup {
		fmt.Printf(" %s => %s\n", k, v)
	}
	for k, f := range fs.files {
		fmt.Printf("+ %s => %s, %d bytes\n", k, f.Media, len(f.Bytes))
	}
}
