package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig holds all configuration for the database connection.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	Log struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
	Database DatabaseConfig `yaml:"database"`
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
