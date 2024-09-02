package storage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"mime"
	"regexp"
	"slices"
)

// TODO: should probably use a library to detect media type from bytes
// e.g. https://pkg.go.dev/github.com/gabriel-vasile/mimetype

// File is a binary object stored in the ProviderStorage. It should be
// referenced by the hash digest returned by Ref().
type File struct {
	Media string // content-type
	Bytes []byte // raw blob
	ref   string
}

// Take a file blob and its content-type, calculate the digest for
// content-addressing and return a *File for the storage.
func NewFile(mimetype string, blob []byte) *File {
	return &File{
		Media: mimetype,
		Bytes: blob,
		ref:   sha256Ref(blob),
	}
}

// Return the sha256: reference of the file.
func (f *File) Ref() string {
	if f.ref == "" {
		f.ref = sha256Ref(f.Bytes)
	}
	return f.ref
}

// sha256Ref takes file's bytes, calculates a SHA256 digest
// and returns a string encoding with hash prefix (sha256:<hex>).
func sha256Ref(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return fmt.Sprintf("sha256:%x", digest)
}

var reSha256Addr = regexp.MustCompile("^sha256:[0-9a-f]{64}$")

// IsRef uses a regular expression to check if the string is a SHA256 content address.
func IsRef(ref string) bool {
	return reSha256Addr.MatchString(ref)
}

var expectedMediaTypes = []string{
	"application/wasm",
	"application/zip",
}

// CheckMediaType tries to parse the given media type, ignoring optional
// params, and checks if it's one of the expected types for our files.
func CheckMediaType(media string) (string, error) {
	mt, _, err := mime.ParseMediaType(media)
	if errors.Is(err, mime.ErrInvalidMediaParameter) {
		err = nil // ignore parameters
	}
	if err != nil {
		return mt, fmt.Errorf("failed parsing: %w", err)
	}
	if !slices.Contains(expectedMediaTypes, mt) {
		err = fmt.Errorf("unexpected media type: %s", mt)
	}
	return mt, err
}
