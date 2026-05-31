package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotenv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nFOO=bar\nQUOTED=\"hello world\"\nEMPTY=\n\nSPACED = spaced \n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FOO", "")           // ensure clean
	_ = os.Unsetenv("FOO")        // not set → should be loaded
	t.Setenv("PRESET", "keepme")  // already set → must not be overridden

	// Add PRESET to the file to prove existing env wins.
	_ = os.WriteFile(path, []byte(content+"PRESET=overwritten\n"), 0o600)

	LoadDotenv(path)

	if got := os.Getenv("FOO"); got != "bar" {
		t.Errorf("FOO = %q, want bar", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello world" {
		t.Errorf("QUOTED = %q, want 'hello world'", got)
	}
	if got := os.Getenv("SPACED"); got != "spaced" {
		t.Errorf("SPACED = %q, want spaced", got)
	}
	if got := os.Getenv("PRESET"); got != "keepme" {
		t.Errorf("PRESET = %q, existing env must win", got)
	}
}

func TestLoadDotenvMissingFileIsNoop(t *testing.T) {
	LoadDotenv("/nonexistent/.env") // must not panic
}
