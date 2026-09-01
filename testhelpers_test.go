package main

import (
	"os"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("could not write test file %s: %v", path, err)
	}
}
