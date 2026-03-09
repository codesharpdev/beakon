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

func TestReadAll_LoadsAllFiles(t *testing.T) {
	root := t.TempDir()
	Init(root)

	files := []pkg.FileIndex{
		{File: "auth/service.go", Hash: "hash1", Symbols: []pkg.BeakonNode{{Name: "Login"}}},
		{File: "api/controller.go", Hash: "hash2", Symbols: []pkg.BeakonNode{{Name: "Handle"}}},
	}
	for _, fi := range files {
		if err := Write(root, fi); err != nil {
			t.Fatal(err)
		}
	}

	all, err := ReadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 file indexes, got %d", len(all))
	}
}

func TestNeedsUpdate_NewFile(t *testing.T) {
	root := t.TempDir()
	Init(root)
	if !NeedsUpdate(root, "auth/service.go", "anyhash") {
		t.Error("expected NeedsUpdate=true for unknown file")
	}
}

func TestNeedsUpdate_SameHash(t *testing.T) {
	root := t.TempDir()
	Init(root)
	fi := pkg.FileIndex{File: "auth/service.go", Hash: "abc123"}
	Write(root, fi)
	if NeedsUpdate(root, "auth/service.go", "abc123") {
		t.Error("expected NeedsUpdate=false when hash unchanged")
	}
}

func TestNeedsUpdate_ChangedHash(t *testing.T) {
	root := t.TempDir()
	Init(root)
	fi := pkg.FileIndex{File: "auth/service.go", Hash: "old"}
	Write(root, fi)
	if !NeedsUpdate(root, "auth/service.go", "new") {
		t.Error("expected NeedsUpdate=true when hash changed")
	}
}

func TestDeleteFile(t *testing.T) {
	root := t.TempDir()
	Init(root)
	fi := pkg.FileIndex{File: "auth/service.go", Hash: "abc"}
	Write(root, fi)
	DeleteFile(root, "auth/service.go")
	if !NeedsUpdate(root, "auth/service.go", "abc") {
		t.Error("expected NeedsUpdate=true after DeleteFile")
	}
}

func TestWriteSymbols_ReadSymbols(t *testing.T) {
	root := t.TempDir()
	Init(root)
	syms := []pkg.BeakonNode{{Name: "Login"}, {Name: "Logout"}}
	if err := WriteSymbols(root, syms); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSymbols(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(got))
	}
}
