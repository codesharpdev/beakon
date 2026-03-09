package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScan_RespectsGitignore(t *testing.T) {
	root := t.TempDir()

	os.MkdirAll(filepath.Join(root, "src"), 0755)
	os.MkdirAll(filepath.Join(root, "generated"), 0755)
	os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(root, "generated", "types.go"), []byte("package gen"), 0644)

	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("generated/\n"), 0644)

	files, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.Contains(f.Path, "generated") {
			t.Errorf("file in ignored dir included: %s", f.Path)
		}
	}

	found := false
	for _, f := range files {
		if strings.Contains(f.Path, "main.go") {
			found = true
		}
	}
	if !found {
		t.Error("src/main.go should be included")
	}
}

func TestScan_NoGitignore_StillWorks(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "src"), 0755)
	os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0644)

	files, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected files to be scanned when no .gitignore exists")
	}
}
