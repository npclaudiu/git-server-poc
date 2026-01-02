package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type IndexStorage struct {
	os       *objectstore.ObjectStore
	repoName string
}

func (s *IndexStorage) SetIndex(idx *index.Index) error {
	var buf bytes.Buffer
	e := index.NewEncoder(&buf)
	if err := e.Encode(idx); err != nil {
		return err
	}

	key := fmt.Sprintf("repositories/%s/index", s.repoName)
	return s.os.Put(context.Background(), key, &buf)
}

func (s *IndexStorage) Index() (*index.Index, error) {
	key := fmt.Sprintf("repositories/%s/index", s.repoName)
	rc, err := s.os.Get(context.Background(), key)
	if err != nil {
		// If no index, return empty index? Or error?
		// go-git usually expects an index if not bare.
		// For bare, maybe it's fine.
		return &index.Index{Version: 2}, nil
	}
	defer rc.Close()

	idx := &index.Index{}
	d := index.NewDecoder(rc)
	if err := d.Decode(idx); err != nil {
		if err == io.EOF {
			return &index.Index{Version: 2}, nil
		}
		return nil, err
	}

	return idx, nil
}
