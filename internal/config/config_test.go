package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/its-the-vibe/eventhorizon/internal/config"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_Defaults(t *testing.T) {
	path := writeYAML(t, `
server:
  host: ""
redis:
  host: "localhost"
  channel: "test"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default server port: got %d, want 8080", cfg.Server.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("default redis port: got %d, want 6379", cfg.Redis.Port)
	}
}

func TestLoad_ExplicitValues(t *testing.T) {
	path := writeYAML(t, `
server:
  host: "127.0.0.1"
  port: 9090
redis:
  host: "redis.example.com"
  port: 6380
  db: 2
  channel: "mychannel"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.Server.Addr(), "127.0.0.1:9090"; got != want {
		t.Errorf("Server.Addr() = %q, want %q", got, want)
	}
	if got, want := cfg.Redis.RedisAddr(), "redis.example.com:6380"; got != want {
		t.Errorf("Redis.RedisAddr() = %q, want %q", got, want)
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeYAML(t, `:::invalid yaml:::`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_DefaultChannel(t *testing.T) {
	path := writeYAML(t, `
server:
  host: ""
redis:
  host: "localhost"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Redis.Channel != "eventhorizon" {
		t.Errorf("default channel = %q, want %q", cfg.Redis.Channel, "eventhorizon")
	}
}

func TestLoad_DefaultLogLevel(t *testing.T) {
	path := writeYAML(t, `
server:
  host: ""
redis:
  host: "localhost"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default log level = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_ExplicitLogLevel(t *testing.T) {
	path := writeYAML(t, `
server:
  host: ""
redis:
  host: "localhost"
log_level: debug
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("log level = %q, want %q", cfg.LogLevel, "debug")
	}
}
