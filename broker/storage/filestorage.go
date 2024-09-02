package storage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"mime"
	"regexp"
)

// TODO: use SQLite, BoltDB or just filesystem for persistence

type FileStorage struct {
	// collection of files in storage. the string here is a content-address, so
	// the files should be effectively deduplicated
	Files map[string]*File
	// a lookup table of plain names to content-addresses
	Lookup map[string]string
}

func NewFileStorage() FileStorage {
	return FileStorage{
		Files:  make(map[string]*File),
		Lookup: make(map[string]string),
	}
}

func (fs *FileStorage) Insert(name, content string, blob []byte) (addr string, err error) {
	// check the content-type first because that's cheapest
	content, err = CheckContentType(content)
	if err != nil {
		return "", fmt.Errorf("parsing content-type failed: %w", err)
	}
	// we could check if the file exists already here but since we operate on
	// memory for now, we can just overwrite whatever is there cheaply
	addr = Address(blob)
	fs.Files[addr] = &File{addr, content, blob}
	// insert in lookup map, if friendly name was given
	if name != "" {
		fs.Lookup[name] = addr
	}
	return
}

type File struct {
	Name    string
	Content string // content-type
	Bytes   []byte
}

// Take a file blob and its content-type, calculate the digest for content
// address and return a *File for the storage.
func NewFile(mimetype string, blob []byte) (*File, error) {
	mt, _, err := mime.ParseMediaType(mimetype)
	if err != nil {
		return nil, err
	}
	file := &File{
		Name:    Address(blob),
		Content: mt,
		Bytes:   blob,
	}
	return file, err
}

// Address takes file contents, calculates a SHA256 digest
// and returns a string encoding with hash prefix (sha256:hex).
func Address(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return fmt.Sprintf("sha256:%x", digest)
}

// IsAddr uses a regexp to check if the string is a sha256: content address.
func IsAddr(name string) bool {
	return reSha256Addr.MatchString(name)
}

var reSha256Addr = regexp.MustCompile("^sha256:[0-9a-f]{64}$")

// CheckContentType tries to parse the given mime type to see
// if it's valid and discards all additional params.
func CheckContentType(content string) (string, error) {
	mt, _, err := mime.ParseMediaType(content)
	if errors.Is(err, mime.ErrInvalidMediaParameter) {
		err = nil // ignore parameters
	}
	return mt, err
}
