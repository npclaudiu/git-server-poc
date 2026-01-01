package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type ObjectStorage struct {
	os       *objectstore.ObjectStore
	repoName string
}

func (s *ObjectStorage) NewEncodedObject() plumbing.EncodedObject {
	return &plumbing.MemoryObject{}
}

func (s *ObjectStorage) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	r, err := obj.Reader()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	defer r.Close()

	h := obj.Hash()
	key := fmt.Sprintf("repositories/%s/objects/%s", s.repoName, h.String())

	// Standard Git loose object header: "type size\0"
	header := fmt.Sprintf("%s %d\000", obj.Type(), obj.Size())
	mr := io.MultiReader(strings.NewReader(header), r)

	if err := s.os.Put(context.Background(), key, mr); err != nil {
		return plumbing.ZeroHash, err
	}

	return h, nil
}

func (s *ObjectStorage) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	key := fmt.Sprintf("repositories/%s/objects/%s", s.repoName, h.String())
	rc, err := s.os.Get(context.Background(), key)
	if err != nil {
		return nil, plumbing.ErrObjectNotFound
	}
	defer rc.Close()

	// Read header
	// We need to read byte by byte until null.
	// Since we can't seek or peek easily on s3 stream without buffering,
	// we'll read a small chunk or byte by byte.

	// Quick hack: Read entire object into memory.
	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	// Find null byte
	nullIdx := -1
	for i, b := range content {
		if b == 0 {
			nullIdx = i
			break
		}
	}

	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid object format: no header")
	}

	header := string(content[:nullIdx])
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid header format")
	}

	objType, err := plumbing.ParseObjectType(parts[0])
	if err != nil {
		return nil, err
	}

	size := int64(0) // parse parts[1]
	fmt.Sscanf(parts[1], "%d", &size)

	o := &plumbing.MemoryObject{}
	o.SetType(objType)
	o.SetSize(size)

	// Write content
	if _, err := o.Write(content[nullIdx+1:]); err != nil {
		return nil, err
	}

	return o, nil
}

func (s *ObjectStorage) IterEncodedObjects(t plumbing.ObjectType) (storer.EncodedObjectIter, error) {
	return nil, nil // Empty iterator?
}

func (s *ObjectStorage) HasEncodedObject(h plumbing.Hash) error {
	key := fmt.Sprintf("repositories/%s/objects/%s", s.repoName, h.String())
	return s.os.Head(context.Background(), key)
}

func (s *ObjectStorage) AddAlternate(remote string) error {
	return nil
}

func (s *ObjectStorage) EncodedObjectSize(h plumbing.Hash) (int64, error) {
	obj, err := s.EncodedObject(plumbing.AnyObject, h)
	if err != nil {
		return 0, err
	}
	return obj.Size(), nil
}
