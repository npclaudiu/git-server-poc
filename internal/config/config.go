package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
	MetaStore struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"meta_store"`
	ObjectStore struct {
		Endpoint        string `yaml:"endpoint"`
		AccessKeyID     string `yaml:"access_key"`
		SecretAccessKey string `yaml:"secret_key"`
		Bucket          string `yaml:"bucket"`
		Region          string `yaml:"region"`
	} `yaml:"object_store"`
}

func Load() (*Config, error) {
	f, err := os.Open("config.yaml")

	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
