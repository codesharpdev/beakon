package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/pkg"
)

func TestWriteIsAtomic(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatal(err)
	}

	fi := pkg.FileIndex{
		File: "auth/service.go",
		Hash: "abc123",
		Symbols: []pkg.BeakonNode{
			{ID: "go:function:auth/service.go:Login", Kind: "function", Name: "Login"},
		},
	}

	if err := Write(root, fi); err != nil {
		t.Fatal(err)
	}

	// Verify no .tmp files remain after a successful write
	entries, _ := os.ReadDir(filepath.Join(root, nodesDir))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("tmp file left behind: %s", e.Name())
		}
	}

	// Verify the file is valid JSON (readable)
	got, err := Read(root, "auth/service.go")
	if err != nil {
		t.Fatalf("read after write: %v", err)
	}
	if got.Hash != "abc123" {
		t.Errorf("got hash %q, want %q", got.Hash, "abc123")
	}
}
