package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
	Redis  RedisConfig  `yaml:"redis"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RedisConfig holds Redis connection configuration.
// Sensitive values (e.g. password) are sourced from environment variables.
type RedisConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	DB      int    `yaml:"db"`
	Channel string `yaml:"channel"`
}

// Load reads a YAML config file and returns a Config.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config %q: %w", path, err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.Redis.Channel == "" {
		cfg.Redis.Channel = "eventhorizon"
	}

	return &cfg, nil
}

// Addr returns the host:port string for the HTTP server.
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// RedisAddr returns the host:port string for Redis.
func (r RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}
