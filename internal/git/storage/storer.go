package storage

import (
	"fmt"

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
		ShallowStorage:   &ShallowStorage{os: os, repoName: repoName},
		ConfigStorage:    &ConfigStorage{os: os, repoName: repoName},
		IndexStorage:     &IndexStorage{os: os, repoName: repoName},
	}
}

func (s *Storer) Module(name string) (storage.Storer, error) {
	return nil, fmt.Errorf("module storage not implemented")
}
