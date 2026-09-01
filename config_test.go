package main

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid config", func(t *testing.T) {
		path := filepath.Join(dir, "valid.yml")
		writeFile(t, path, `
proto: 100
pipes:
  - name: pipe1
    prefix: 10.0.0.0/8
    source: 254
    target: 100
  - name: pipe2
    prefix: 0.0.0.0/0
    source: 254
    target: 200
`)

		cfg, err := loadConfig(path)
		if err != nil {
			t.Fatalf("loadConfig() returned unexpected error: %v", err)
		}

		if cfg.Proto != 100 {
			t.Errorf("cfg.Proto = %d, want 100", cfg.Proto)
		}

		if len(cfg.Pipes) != 2 {
			t.Fatalf("len(cfg.Pipes) = %d, want 2", len(cfg.Pipes))
		}

		p := cfg.Pipes[0]
		if p.Name != "pipe1" || p.Prefix != "10.0.0.0/8" || p.Source != 254 || p.Target != 100 {
			t.Errorf("cfg.Pipes[0] = %+v, unexpected values", p)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadConfig(filepath.Join(dir, "does-not-exist.yml"))
		if err == nil {
			t.Fatal("loadConfig() expected an error for a missing file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := filepath.Join(dir, "invalid.yml")
		writeFile(t, path, `proto: [this is not valid: yaml`)

		_, err := loadConfig(path)
		if err == nil {
			t.Fatal("loadConfig() expected an error for invalid yaml, got nil")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		path := filepath.Join(dir, "empty.yml")
		writeFile(t, path, ``)

		cfg, err := loadConfig(path)
		if err != nil {
			t.Fatalf("loadConfig() returned unexpected error for empty file: %v", err)
		}

		if len(cfg.Pipes) != 0 {
			t.Errorf("len(cfg.Pipes) = %d, want 0 for empty config", len(cfg.Pipes))
		}
	})
}
