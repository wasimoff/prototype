package storage

import (
	"errors"
	"fmt"
	"iter"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

type BoltFileStorage struct {
	db *bolt.DB
}

var (
	fileBucket      = []byte("files")
	mediaTypeBucket = []byte("mediatypes")
	lookupBucket    = []byte("lookup")
)

func NewBoltFileStorage(path string) *FileStorage {

	// open the boltdb file
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		// to keep the API clean, we just abort in here since this happens only at startup
		log.Fatalf("boltfs: cannot open db: %s", err)
	}

	// ensure that all buckets exist
	err = db.Update(func(tx *bolt.Tx) (err error) {
		if _, e := tx.CreateBucketIfNotExists(fileBucket); e != nil {
			errors.Join(err, e)
		}
		if _, e := tx.CreateBucketIfNotExists(mediaTypeBucket); e != nil {
			errors.Join(err, e)
		}
		if _, e := tx.CreateBucketIfNotExists(lookupBucket); e != nil {
			errors.Join(err, e)
		}
		return
	})
	if err != nil {
		log.Fatalf("boltfs: cannot create buckets: %s", err)
	}

	return &FileStorage{
		AbstractFileStorage: &BoltFileStorage{db},
	}
}

// Insert a new file into the Storage. The optional `name` will be inserted
// into the lookup table and can be used to resolve the file later.
func (fs *BoltFileStorage) Insert(name, media string, blob []byte) (file *File, err error) {

	// check the media type first because that's cheapest
	media, err = CheckMediaType(media)
	if err != nil {
		return nil, fmt.Errorf("media: %w", err)
	}

	// use a *File struct to obtain the content hash
	file = NewFile(media, blob)
	ref := file.Ref()

	err = fs.db.Update(func(tx *bolt.Tx) error {
		// insert blob and mediatype into buckets
		if err := tx.Bucket(fileBucket).Put([]byte(ref), blob); err != nil {
			return err
		}
		if err := tx.Bucket(mediaTypeBucket).Put([]byte(ref), []byte(media)); err != nil {
			return err
		}
		// insert name in lookup, if given
		if name != "" {
			if err := tx.Bucket(lookupBucket).Put([]byte(name), []byte(ref)); err != nil {
				return err
			}
		}
		return nil
	})
	return file, err

}

// Get a File from Storage, either by Ref or a friendly name in lookup map.
func (fs *BoltFileStorage) Get(nameOrRef string) (f *File) {
	// attempt to fetch by ref directly
	f = fs.get(nameOrRef)
	if f == nil {
		// try to lookup a friendly name
		fs.db.View(func(tx *bolt.Tx) error {
			ref := tx.Bucket(lookupBucket).Get([]byte(nameOrRef))
			f = fs.get(string(ref))
			return nil
		})
	}
	// at this point f is either nil or successfully resolved ...
	return
}

func (fs *BoltFileStorage) get(ref string) (f *File) {
	fs.db.View(func(tx *bolt.Tx) error {
		// try to get the file value and mediatype
		value := tx.Bucket(fileBucket).Get([]byte(ref))
		media := tx.Bucket(mediaTypeBucket).Get([]byte(ref))
		if value == nil || media == nil {
			return nil // no such file, f is nil
		}
		// copy the blob and return a *File
		blob := make([]byte, len(value))
		copy(blob, value)
		f = NewFile(string(media), blob)
		return nil // ok
	})
	return
}

// Iterator over all Files in the storage.
func (fs *BoltFileStorage) All() iter.Seq2[string, *File] {
	return func(yield func(string, *File) bool) {
		fs.db.View(func(tx *bolt.Tx) error {
			return tx.Bucket(fileBucket).ForEach(func(k, _ []byte) error {
				ref := string(k)
				file := fs.get(ref)
				if file == nil {
					panic("boltfs: got a nil *File while iterating in All()")
				}
				if !yield(ref, file) {
					return errors.New("end iteration")
				}
				return nil
			})
		})
	}
}
