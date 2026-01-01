package storage

import (
	"github.com/go-git/go-git/v5/config"
)

type ConfigStorage struct{}

func (s *ConfigStorage) Config() (*config.Config, error)  { return config.NewConfig(), nil }
func (s *ConfigStorage) SetConfig(c *config.Config) error { return nil }
