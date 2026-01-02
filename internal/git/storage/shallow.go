package storage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type ShallowStorage struct {
	os       *objectstore.ObjectStore
	repoName string
}

func (s *ShallowStorage) SetShallow(commits []plumbing.Hash) error {
	var buf bytes.Buffer
	for _, h := range commits {
		buf.WriteString(h.String() + "\n")
	}

	key := fmt.Sprintf("repositories/%s/shallow", s.repoName)
	return s.os.Put(context.Background(), key, &buf)
}

func (s *ShallowStorage) Shallow() ([]plumbing.Hash, error) {
	key := fmt.Sprintf("repositories/%s/shallow", s.repoName)
	rc, err := s.os.Get(context.Background(), key)
	if err != nil {
		// If no shallow file, return empty list (not error)
		return nil, nil
	}
	defer rc.Close()

	var hashes []plumbing.Hash
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		hashes = append(hashes, plumbing.NewHash(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hashes, nil
}
