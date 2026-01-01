package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type Storer struct {
	*ObjectStorage
	*ReferenceStorage
	*ShallowStorage
	*ConfigStorage
	*IndexStorage
}

func NewStorer(os *objectstore.ObjectStore, ms *metastore.MetaStore, repoName string) *Storer {
	return &Storer{
		ObjectStorage:    &ObjectStorage{os: os, repoName: repoName},
		ReferenceStorage: &ReferenceStorage{ms: ms, repoName: repoName},
		ShallowStorage:   &ShallowStorage{},
		ConfigStorage:    &ConfigStorage{},
		IndexStorage:     &IndexStorage{},
	}
}

func (s *Storer) Module(name string) (storage.Storer, error) {
	return nil, fmt.Errorf("module storage not implemented")
}

// ObjectStorage

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

// ReferenceStorage
type ReferenceStorage struct {
	ms       *metastore.MetaStore
	repoName string
}

func (s *ReferenceStorage) SetReference(ref *plumbing.Reference) error {
	target := ""
	if ref.Type() == plumbing.SymbolicReference {
		target = ref.Target().String()
	}
	// For HashReference, ref.Hash().String()
	// For Symbolic, ref.Target().String(), hash is empty?
	hash := ""
	if ref.Type() == plumbing.HashReference {
		hash = ref.Hash().String()
	}

	err := s.ms.PutRef(context.Background(), s.repoName, ref.Name().String(), ref.Type().String(), hash, target)
	return err
}

func (s *ReferenceStorage) CheckAndSetReference(new, old *plumbing.Reference) error {
	// Simple optimistic lock? Or just overwrite?
	// For PoC, just Set.
	return s.SetReference(new)
}

func (s *ReferenceStorage) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	ref, err := s.ms.GetRef(context.Background(), s.repoName, n.String())
	if err != nil {
		return nil, plumbing.ErrReferenceNotFound
	}

	if ref.Type == "symbolic" { // string "symbolic"
		return plumbing.NewSymbolicReference(n, plumbing.ReferenceName(ref.Target.String)), nil
	}
	return plumbing.NewHashReference(n, plumbing.NewHash(ref.Hash.String)), nil
}

func (s *ReferenceStorage) IterReferences() (storer.ReferenceIter, error) {
	refs, err := s.ms.ListRefs(context.Background(), s.repoName)
	if err != nil {
		return nil, err
	}
	// Convert to iterator
	var r []*plumbing.Reference
	for _, ref := range refs {
		if ref.Type == "symbolic" {
			r = append(r, plumbing.NewSymbolicReference(plumbing.ReferenceName(ref.RefName), plumbing.ReferenceName(ref.Target.String)))
		} else {
			r = append(r, plumbing.NewHashReference(plumbing.ReferenceName(ref.RefName), plumbing.NewHash(ref.Hash.String)))
		}
	}
	return storer.NewReferenceSliceIter(r), nil
}

func (s *ReferenceStorage) RemoveReference(n plumbing.ReferenceName) error {
	return s.ms.DeleteRef(context.Background(), s.repoName, n.String())
}

func (s *ReferenceStorage) CountLooseRefs() (int, error) {
	return 0, nil
}

func (s *ReferenceStorage) PackRefs() error {
	return nil
}

// Stubs for others
type ShallowStorage struct{}

func (s *ShallowStorage) SetShallow(commits []plumbing.Hash) error { return nil }
func (s *ShallowStorage) Shallow() ([]plumbing.Hash, error)        { return nil, nil }

type ConfigStorage struct{}

func (s *ConfigStorage) Config() (*config.Config, error)  { return config.NewConfig(), nil }
func (s *ConfigStorage) SetConfig(c *config.Config) error { return nil }

type IndexStorage struct{}

func (s *IndexStorage) SetIndex(index *index.Index) error { return nil }
func (s *IndexStorage) Index() (*index.Index, error)      { return nil, nil }
