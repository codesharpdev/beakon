package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/internal/context"
	"github.com/beakon/beakon/internal/index"
)

func TestFullPipeline_IndexAndQuery(t *testing.T) {
	root := t.TempDir()
	copyDir(t, "../../testdata/sample_repo", root)

	// 1. Full index
	result, err := Run(root)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Errors) > 0 {
		t.Fatalf("indexer errors: %v", result.Errors)
	}
	if result.SymbolsFound == 0 {
		t.Error("expected at least 1 symbol indexed")
	}

	// 2. Symbols persisted
	syms, err := index.ReadSymbols(root)
	if err != nil {
		t.Fatalf("ReadSymbols: %v", err)
	}
	found := false
	for _, s := range syms {
		if s.Name == "AuthService.Login" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AuthService.Login not found in symbols index")
	}

	// 3. Context engine returns a bundle
	engine := context.NewEngine(root)
	bundle, err := engine.Assemble("AuthService.Login")
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if bundle.Anchor.Symbol != "AuthService.Login" {
		t.Errorf("anchor = %q, want AuthService.Login", bundle.Anchor.Symbol)
	}
	if len(bundle.Callees) == 0 {
		t.Error("expected at least 1 callee for AuthService.Login")
	}

	// 4. Incremental update — touch the file
	svcPath := filepath.Join(root, "auth", "service.go")
	src, _ := os.ReadFile(svcPath)
	os.WriteFile(svcPath, append(src, []byte("\n// touch\n")...), 0644)

	ur, err := UpdateFile(root, svcPath)
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}
	if ur.Skipped {
		t.Error("expected file to be re-indexed after change")
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
}
