package storage

import (
	"crypto/sha256"
	"time"
)

type File struct {
	Name   string   `msgpack:"filename"`
	Hash   [32]byte `msgpack:"hash"`
	Bytes  []byte   `msgpack:"bytes,omitempty"`
	Length int      `msgpack:"length"`
	Epoch  int64    `msgpack:"epoch"`
}

func (f *File) CloneWithoutBytes() *File {
	return &File{f.Name, f.Hash, nil, f.Length, f.Epoch}
}

type FileStorage struct {
	// TODO: use SQLite or BoltDB database for persistence
	Files map[string]*File
}

func NewFileStorage() FileStorage {
	return FileStorage{
		Files: make(map[string]*File),
	}
}

func NewFile(name string, buf []byte) *File {
	return &File{
		Name:   name,
		Hash:   filehash(buf),
		Bytes:  buf,
		Length: len(buf),
		Epoch:  time.Now().UnixMilli(),
	}
}

func (fs *FileStorage) Insert(file *File) {
	fs.Files[file.Name] = file
}

func filehash(buf []byte) [32]byte {
	return sha256.Sum256(buf)
}
