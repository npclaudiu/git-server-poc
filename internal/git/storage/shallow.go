package storage

import (
	"github.com/go-git/go-git/v5/plumbing"
)

type ShallowStorage struct{}

func (s *ShallowStorage) SetShallow(commits []plumbing.Hash) error { return nil }
func (s *ShallowStorage) Shallow() ([]plumbing.Hash, error)        { return nil, nil }
