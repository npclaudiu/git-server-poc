package storage

import (
	"github.com/go-git/go-git/v5/plumbing/format/index"
)

type IndexStorage struct{}

func (s *IndexStorage) SetIndex(index *index.Index) error { return nil }
func (s *IndexStorage) Index() (*index.Index, error)      { return nil, nil }
