package main

import (
	"path/filepath"
	"testing"
)

func TestLoadPipesFromConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid config produces matching pipes", func(t *testing.T) {
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

		pipes, err := loadPipesFromConfig(path)
		if err != nil {
			t.Fatalf("loadPipesFromConfig() returned unexpected error: %v", err)
		}

		if len(pipes) != 2 {
			t.Fatalf("len(pipes) = %d, want 2", len(pipes))
		}

		if pipes[0].name != "pipe1" {
			t.Errorf("pipes[0].name = %q, want %q", pipes[0].name, "pipe1")
		}

		if pipes[0].sourceTable != 254 || pipes[0].targetTable != 100 {
			t.Errorf("pipes[0] tables = (%d, %d), want (254, 100)", pipes[0].sourceTable, pipes[0].targetTable)
		}

		// 0.0.0.0/0 is the default route, newPipe() should turn it into a nil prefix.
		if pipes[1].prefix != nil {
			t.Errorf("pipes[1].prefix = %v, want nil for default route", pipes[1].prefix)
		}
	})

	t.Run("invalid prefix returns error", func(t *testing.T) {
		path := filepath.Join(dir, "invalid-prefix.yml")
		writeFile(t, path, `
proto: 100
pipes:
  - name: pipe1
    prefix: not-a-cidr
    source: 254
    target: 100
`)

		_, err := loadPipesFromConfig(path)
		if err == nil {
			t.Fatal("loadPipesFromConfig() expected an error for an invalid prefix, got nil")
		}
	})

	t.Run("missing config file returns error", func(t *testing.T) {
		_, err := loadPipesFromConfig(filepath.Join(dir, "does-not-exist.yml"))
		if err == nil {
			t.Fatal("loadPipesFromConfig() expected an error for a missing config file, got nil")
		}
	})

	t.Run("empty config returns no pipes", func(t *testing.T) {
		path := filepath.Join(dir, "empty.yml")
		writeFile(t, path, `proto: 100`)

		pipes, err := loadPipesFromConfig(path)
		if err != nil {
			t.Fatalf("loadPipesFromConfig() returned unexpected error: %v", err)
		}

		if len(pipes) != 0 {
			t.Errorf("len(pipes) = %d, want 0", len(pipes))
		}
	})
}
