package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirExists(t *testing.T) {
	if DirExists("/nonexistent-path-xyz") {
		t.Fatal("expected false")
	}
	tmp := t.TempDir()
	if DirExists(tmp) {
		t.Fatal("empty dir should be false")
	}
	sqlPath := filepath.Join(tmp, "001_a.sql")
	if err := os.WriteFile(sqlPath, []byte("-- x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !DirExists(tmp) {
		t.Fatal("expected true when sql present")
	}
}
