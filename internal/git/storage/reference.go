package storage

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
)

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
