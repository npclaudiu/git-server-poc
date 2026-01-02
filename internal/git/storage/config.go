package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/config"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type ConfigStorage struct {
	os       *objectstore.ObjectStore
	repoName string
}

func (s *ConfigStorage) Config() (*config.Config, error) {
	key := fmt.Sprintf("repositories/%s/config", s.repoName)
	rc, err := s.os.Get(context.Background(), key)
	if err != nil {
		// If config doesn't exist, return new empty config
		return config.NewConfig(), nil
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	cfg := config.NewConfig()
	if err := cfg.Unmarshal(content); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (s *ConfigStorage) SetConfig(c *config.Config) error {
	content, err := c.Marshal()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("repositories/%s/config", s.repoName)
	return s.os.Put(context.Background(), key, bytes.NewReader(content))
}
