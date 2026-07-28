package init

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldDoesNotOverwriteExistingFiles(t *testing.T) {
	target := t.TempDir()
	configPath := filepath.Join(target, "_config.example.yaml")

	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	err := Scaffold(target)
	if err == nil {
		t.Fatal("expected scaffold to reject overwriting existing files")
	}

	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read existing file: %v", readErr)
	}
	if string(data) != "existing" {
		t.Fatalf("expected existing file to stay untouched, got %q", string(data))
	}
}
