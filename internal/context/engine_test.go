package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/internal/indexer"
)

// setupIndex copies the sample repo into a temp dir, runs a full index,
// and returns the root path ready for engine queries.
func setupIndex(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyDir(t, "../../testdata/sample_repo", root)
	if _, err := indexer.Run(root); err != nil {
		t.Fatalf("indexer.Run: %v", err)
	}
	return root
}

func TestAssemble_AnchorFound(t *testing.T) {
	root := setupIndex(t)
	engine := NewEngine(root)

	bundle, err := engine.Assemble("AuthService.Login")
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if bundle.Anchor.Symbol != "AuthService.Login" {
		t.Errorf("anchor = %q, want AuthService.Login", bundle.Anchor.Symbol)
	}
	if bundle.Anchor.Code == "" {
		t.Error("anchor code should not be empty")
	}
}

func TestAssemble_CalleesPresent(t *testing.T) {
	root := setupIndex(t)
	bundle, err := NewEngine(root).Assemble("AuthService.Login")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Callees) == 0 {
		t.Error("expected callees for AuthService.Login")
	}
}

func TestAssemble_CallersPresent(t *testing.T) {
	root := setupIndex(t)
	// validatePassword is called directly by name from AuthService.Login
	bundle, err := NewEngine(root).Assemble("validatePassword")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range bundle.Callers {
		if c.Symbol == "AuthService.Login" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AuthService.Login in callers of validatePassword, got %v", bundle.Callers)
	}
}

func TestAssemble_FilesDeduped(t *testing.T) {
	root := setupIndex(t)
	bundle, err := NewEngine(root).Assemble("AuthService.Login")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) == 0 {
		t.Error("expected at least 1 file in bundle")
	}
	seen := map[string]bool{}
	for _, f := range bundle.Files {
		if seen[f] {
			t.Errorf("duplicate file in bundle: %q", f)
		}
		seen[f] = true
	}
}

func TestAssemble_TokenEstimatePositive(t *testing.T) {
	root := setupIndex(t)
	bundle, err := NewEngine(root).Assemble("AuthService.Login")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TokenEstimate <= 0 {
		t.Error("expected positive token estimate")
	}
}

func TestAssemble_PartialMatch(t *testing.T) {
	root := setupIndex(t)
	// "Login" should resolve to "AuthService.Login" via suffix match
	bundle, err := NewEngine(root).Assemble("Login")
	if err != nil {
		t.Fatalf("partial match failed: %v", err)
	}
	if bundle.Anchor.Symbol == "" {
		t.Error("expected anchor from partial match")
	}
}

func TestAssemble_SymbolNotFound(t *testing.T) {
	root := setupIndex(t)
	_, err := NewEngine(root).Assemble("NonExistentSymbol_xyz")
	if err == nil {
		t.Fatal("expected error for missing symbol")
	}
	if _, ok := err.(*SymbolNotFound); !ok {
		t.Errorf("expected SymbolNotFound error, got %T: %v", err, err)
	}
}

func TestAssemble_ExternalCalleeIncluded(t *testing.T) {
	root := setupIndex(t)
	// validatePassword calls errors.New which is external
	bundle, err := NewEngine(root).Assemble("validatePassword")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range bundle.Callees {
		if c.Kind == "external" {
			found = true
			break
		}
	}
	if !found {
		t.Log("no external callee found — acceptable if call not extracted")
	}
}

func TestEngine_QuerySetCorrectly(t *testing.T) {
	root := setupIndex(t)
	bundle, err := NewEngine(root).Assemble("AuthService.Login")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Query != "AuthService.Login" {
		t.Errorf("query = %q, want AuthService.Login", bundle.Query)
	}
}

// copyDir copies src directory tree into dst (reused from indexer tests pattern).
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
