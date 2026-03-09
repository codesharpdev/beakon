package indexer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentUpdateFileDoesNotRace(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "auth")
	os.MkdirAll(srcDir, 0755)

	src := []byte(`package auth

func Login() {}
func Logout() {}
`)
	srcPath := filepath.Join(srcDir, "service.go")
	os.WriteFile(srcPath, src, 0644)

	// Full index first
	result, err := Run(root)
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("initial index failed: %v %v", err, result.Errors)
	}

	// Concurrent updates to the same file must not race
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			UpdateFile(root, srcPath)
		}()
	}
	wg.Wait()
	// If -race detector fires, the test fails. No assertion needed.
}
