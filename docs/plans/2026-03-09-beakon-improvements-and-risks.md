# Beakon — Improvements & Risk Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the project from CodeIndex to Beakon (including storage directory rename `files/` → `nodes/`), fix all partially-implemented and missing capabilities identified in the capability audit, then implement the top improvements that move it toward production readiness for AI agent use.

**Architecture:** Local-first OSS. All data stays in `.beakon/` in the repo root. No server, no daemon, no network. Every fix must maintain the stateless, JSON-only, query-from-disk design.

**Tech Stack:** Go 1.22, Tree-sitter (go-tree-sitter), Cobra CLI, fsnotify — no new dependencies allowed.

---

## PART A — PROJECT RENAME: CodeIndex → Beakon

All references to `codeindex`, `CodeIndex`, `.codeindex` must become `beakon`, `Beakon`, `.beakon`.
This is a mechanical rename. Do it in one commit before touching any logic.

---

### Task 1: Rename go module path and all import paths

**Files:**
- Modify: `go.mod:1`
- Modify: `internal/index/write.go:12-13`
- Modify: `internal/indexer/update.go:9-12`
- Modify: `internal/indexer/index_repo.go` (all imports)
- Modify: `internal/indexer/watch.go` (no internal imports currently)
- Modify: `internal/graph/build.go:9`
- Modify: `internal/context/engine.go` (all imports)
- Modify: `internal/symbols/extract.go:10`
- Modify: `internal/symbols/parse.go` (all imports)
- Modify: `internal/repo/scan.go` (all imports)
- Modify: `internal/code/fetch.go` (all imports)
- Modify: `cmd/codeindex/main.go` (all imports)

**Step 1: Update go.mod**

Change line 1 from:
```
module github.com/codeindex/codeindex
```
to:
```
module github.com/beakon/beakon
```

**Step 2: Replace all import paths in every .go file**

Run this to find all occurrences:
```bash
grep -r "github.com/codeindex/codeindex" --include="*.go" -l
```

In every file found, replace `github.com/codeindex/codeindex` with `github.com/beakon/beakon`.

**Step 3: Verify the build compiles**

```bash
go build ./...
```
Expected: no errors.

**Step 4: Commit**

```bash
git add go.mod $(find . -name "*.go" | xargs grep -l "beakon/beakon")
git commit -m "chore: rename module path to github.com/beakon/beakon"
```

---

### Task 2: Rename binary and CLI command directory

**Files:**
- Rename: `cmd/codeindex/` → `cmd/beakon/`
- Modify: `CLAUDE.md` (all `./codeindex` → `./beakon`)

**Step 1: Move the directory**

```bash
mv cmd/codeindex cmd/beakon
```

**Step 2: Verify build still works**

```bash
go build -o beakon ./cmd/beakon
```
Expected: binary `./beakon` is created.

**Step 3: Update CLAUDE.md**

Find every occurrence of `./codeindex` or `codeindex` in CLAUDE.md and replace with `./beakon` or `beakon`. The table of commands, the build instructions, and the First Thing Every Session section all reference the old binary name.

**Step 4: Commit**

```bash
git add cmd/beakon CLAUDE.md
git commit -m "chore: rename CLI binary and command directory to beakon"
```

---

### Task 3: Rename storage directory .codeindex → .beakon and files/ → nodes/

The storage directory name is hardcoded in two files, and the `files/` subdirectory — which stores one `FileIndex` (containing `[]CodeIndexNode`) per source file — should be named `nodes/` to reflect what it actually stores.

```
OLD layout:              NEW layout:
.codeindex/              .beakon/
  meta.json               meta.json
  symbols.json            symbols.json
  map.json                map.json
  files/           →      nodes/
    auth_service.json       auth_service.json
  graph/                  graph/
    calls_from.json         calls_from.json
    calls_to.json           calls_to.json
```

**Files to modify:**
- `internal/index/write.go:15-18` — constants `codeindexDir`, `filesDir`
- `internal/graph/build.go:12` — constant `graphDir`
- `internal/repo/scan.go` — skip list entry `".codeindex"`
- `internal/indexer/watch.go:165-173` — `skipWatchDirs` map entry `".codeindex"`
- `internal/indexer/update.go` — no direct path references (inherits from index package)

**Step 1: Update constants in index/write.go**

Current (lines 15-18):
```go
const (
	codeindexDir = ".codeindex"
	filesDir     = ".codeindex/files"
)
```

Change to:
```go
const (
	beakonDir = ".beakon"
	nodesDir  = ".beakon/nodes"
)
```

Then:
- Replace every use of `codeindexDir` in the file with `beakonDir`
- Replace every use of `filesDir` in the file with `nodesDir`

**Step 2: Update Init() path string in index/write.go**

Current (lines 143-153):
```go
func Init(root string) error {
	dirs := []string{
		filepath.Join(root, codeindexDir),
		filepath.Join(root, filesDir),
		filepath.Join(root, ".codeindex/graph"),
	}
```

Change to:
```go
func Init(root string) error {
	dirs := []string{
		filepath.Join(root, beakonDir),
		filepath.Join(root, nodesDir),
		filepath.Join(root, ".beakon/graph"),
	}
```

**Step 3: Update constant in graph/build.go**

Current (line 12):
```go
const graphDir = ".codeindex/graph"
```

Change to:
```go
const graphDir = ".beakon/graph"
```

**Step 4: Update skip lists**

In `internal/repo/scan.go`, find the ignored-directories slice and replace `".codeindex"` with `".beakon"`.

In `internal/indexer/watch.go` (lines 165-173), update `skipWatchDirs`:
```go
var skipWatchDirs = map[string]bool{
    ".git":         true,
    ".beakon":      true,   // was .codeindex
    "node_modules": true,
    "vendor":       true,
    "dist":         true,
    "build":        true,
    "__pycache__":  true,
}
```

**Step 5: Verify build and quick smoke test**

```bash
go build -o beakon ./cmd/beakon
./beakon index
ls .beakon/        # meta.json  symbols.json  map.json  nodes/  graph/
ls .beakon/nodes/  # auth_service.go.json  api_controller.go.json
```

Expected: `.beakon/nodes/` exists (not `files/`). Old `.codeindex/` directory is not created.

**Step 6: Commit**

```bash
git add internal/index/write.go internal/graph/build.go internal/repo/scan.go internal/indexer/watch.go
git commit -m "chore: rename storage dirs .codeindex→.beakon and files/→nodes/"
```

---

### Task 4: Update all documentation and string references

**Files:**
- `SPEC.md` — all `.codeindex` path references
- `ARCHITECTURE.md` — all references
- `TASKS.md` — all references
- `REPO_RULES.md` — all references
- `TESTING.md` — all references
- `cmd/beakon/main.go` — any hardcoded string `"codeindex"` in help text or output

**Step 1: Search for remaining references**

```bash
grep -r "codeindex\|CodeIndex" --include="*.md" -l
grep -r "codeindex\|CodeIndex" --include="*.go" -l
```

**Step 2: Update each file**

For every `.md` file found:
- Replace `.codeindex/` with `.beakon/`
- Replace `codeindex` (binary name) with `beakon`
- Replace `CodeIndex` (project name) with `Beakon`

For Go files: check help text strings, error messages, and comments. Update any that say "codeindex" or "CodeIndex". Known location:

`internal/context/engine.go:217` has a hardcoded error message:
```go
return "symbol not found: " + e.Name + " — run 'codeindex index' first"
```
Change to:
```go
return "symbol not found: " + e.Name + " — run 'beakon index' first"
```

Do NOT rename Go package names (e.g. `package index`, `package symbols`) — those follow directory names, not the binary name.

**Step 3: Verify**

```bash
grep -r "codeindex\|CodeIndex" --include="*.go" --include="*.md" .
```
Expected: zero results (or only inside the git history).

**Step 4: Commit**

```bash
git add -p   # review each change
git commit -m "chore: update all documentation and help text to Beakon"
```

---

## PART B — RISK FIXES

---

### Task 5: Fix atomic writes (Risk #2 — HIGH)

**Problem:** `writeJSON` in both `internal/index/write.go:165-177` and `internal/graph/build.go:220-229` write directly to the target file. A crash mid-write leaves corrupted JSON. The fix: write to a `.tmp` file then atomically rename it over the target.

**Files:**
- Modify: `internal/index/write.go:165-177`
- Modify: `internal/graph/build.go:220-229`

**Step 1: Write the failing test for index writes**

Create file `internal/index/write_test.go`:
```go
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
		Symbols: []pkg.CodeIndexNode{
			{ID: "go:function:auth/service.go:Login", Kind: "function", Name: "Login"},
		},
	}

	if err := Write(root, fi); err != nil {
		t.Fatal(err)
	}

	// Verify no .tmp files remain after a successful write
	entries, _ := os.ReadDir(filepath.Join(root, filesDir))
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
```

**Step 2: Run test to verify it passes (baseline)**

```bash
go test ./internal/index/... -v -run TestWriteIsAtomic
```
Expected: PASS (the test only checks no .tmp files remain, so it passes against current code).

**Step 3: Replace writeJSON in internal/index/write.go**

Current `writeJSON` (lines 165-177):
```go
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
```

Replace with:
```go
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
```

**Step 4: Replace writeJSON in internal/graph/build.go**

Current `writeJSON` (lines 220-229):
```go
func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
```

Replace with (same pattern as above):
```go
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
```

Note: `os` and `filepath` are already imported in both files.

**Step 5: Run test again**

```bash
go test ./internal/index/... -v -run TestWriteIsAtomic
go build ./...
```
Expected: all pass, build clean.

**Step 6: Commit**

```bash
git add internal/index/write.go internal/graph/build.go internal/index/write_test.go
git commit -m "fix: atomic writes in index and graph using tmp+rename"
```

---

### Task 6: Fix concurrent write safety in watch mode (Risk #3 — HIGH)

**Problem:** `rebuildGlobal()` in `internal/indexer/update.go:131` is called from the file watcher goroutine. If two file events are processed concurrently (which the current watcher prevents via its single-threaded event loop, but `UpdateFile` can be called externally), there is no mutex protecting the global writes. The safe fix: add a package-level mutex around `rebuildGlobal`.

**Files:**
- Modify: `internal/indexer/update.go`

**Step 1: Add a package-level mutex**

At the top of `internal/indexer/update.go`, after the `import` block, add:

```go
// globalMu serializes all writes to symbols.json, map.json, and graph files.
// Multiple file updates must not rebuild global indexes concurrently.
var globalMu sync.Mutex
```

Also add `"sync"` to the imports if not already present (check: current imports are `fmt`, `os`, `path/filepath`, `time`).

**Step 2: Lock the mutex in UpdateFile before calling rebuildGlobal**

In `UpdateFile` (line 93), the call is:
```go
if err := rebuildGlobal(root, rel, newSyms, newCalls); err != nil {
```

Wrap it:
```go
globalMu.Lock()
err = rebuildGlobal(root, rel, newSyms, newCalls)
globalMu.Unlock()
if err != nil {
    return nil, fmt.Errorf("rebuild global: %w", err)
}
```

Also in `removeFile` (line 116):
```go
if err := rebuildGlobal(root, rel, nil, nil); err != nil {
```

Wrap it the same way.

**Step 3: Write a test proving sequential behavior**

Add to a new file `internal/indexer/update_test.go`:
```go
package indexer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentUpdateFileDoesNotRace(t *testing.T) {
	// Create a minimal repo with a beakon index
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
```

**Step 4: Run with race detector**

```bash
go test -race ./internal/indexer/... -run TestConcurrentUpdateFileDoesNotRace -v
```
Expected: PASS with no race condition detected.

**Step 5: Commit**

```bash
git add internal/indexer/update.go internal/indexer/update_test.go
git commit -m "fix: serialize rebuildGlobal with mutex to prevent write races"
```

---

### Task 7: Fix signal handling and graceful shutdown in watch mode (Risk — MEDIUM)

**Problem:** `internal/indexer/watch.go` has a commented-out signal handler. The `watch` command in `cmd/beakon/main.go` blocks forever on the event loop with no Ctrl+C handler.

**Files:**
- Modify: `cmd/beakon/main.go` — the `watch` command's Run function

**Step 1: Find the watch command in main.go**

Search for `watchCmd` in `cmd/beakon/main.go`. The Run function currently calls `w.Start()` which blocks.

**Step 2: Add signal handling**

Replace the blocking `w.Start()` call with a goroutine + signal wait:

```go
// At the top of the file, ensure these imports exist:
// "os/signal"
// "syscall"

// Inside the watch command's Run function, replace w.Start() with:
go w.Start()

quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

for {
    select {
    case <-quit:
        fmt.Println("\nShutting down watcher...")
        w.Stop()
        return nil
    case ev, ok := <-w.Events:
        if !ok {
            return nil
        }
        if ev.Err != nil {
            fmt.Fprintf(os.Stderr, "watch error: %v\n", ev.Err)
            continue
        }
        if ev.Result != nil && !ev.Result.Skipped {
            if human {
                fmt.Printf("updated %s (%d symbols, %s)\n",
                    ev.Result.FilePath, ev.Result.SymbolsAfter, ev.Result.Duration.Round(time.Millisecond))
            } else {
                printJSON(ev.Result)
            }
        }
    }
}
```

**Step 3: Build and manually test**

```bash
go build -o beakon ./cmd/beakon
./beakon watch
# Press Ctrl+C
```
Expected: prints "Shutting down watcher..." and exits cleanly (exit code 0).

**Step 4: Commit**

```bash
git add cmd/beakon/main.go
git commit -m "fix: graceful Ctrl+C shutdown in watch mode with signal handling"
```

---

### Task 8: Add panic recovery to indexer worker pool (Risk — MEDIUM)

**Problem:** Worker goroutines in `internal/indexer/index_repo.go` have no `recover()`. A panic from a malformed source file silently exits the goroutine.

**Files:**
- Modify: `internal/indexer/index_repo.go`

**Step 1: Find the worker goroutine**

In `index_repo.go`, find the goroutine that calls `symbols.Extract`. It looks like:
```go
go func(sf repo.SourceFile) {
    defer wg.Done()
    // ... parse and write ...
}(sf)
```

**Step 2: Add recover inside the goroutine**

Wrap the goroutine body with a recover that converts panics into errors:
```go
go func(sf repo.SourceFile) {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            mu.Lock()
            errs = append(errs, fmt.Errorf("panic processing %s: %v", sf.Path, r))
            mu.Unlock()
        }
    }()
    // ... existing body unchanged ...
}(sf)
```

`mu` and `errs` are the existing mutex and error slice already in the function — check the exact variable names in `index_repo.go` and use the same ones.

**Step 3: Verify build**

```bash
go build ./...
```
Expected: no errors.

**Step 4: Commit**

```bash
git add internal/indexer/index_repo.go
git commit -m "fix: recover from panics in indexer worker goroutines"
```

---

## PART C — IMPROVEMENTS

---

### Task 9: Test suite — symbol extraction

This is the most foundational test. It uses the existing `testdata/sample_repo`.

**Files:**
- Create: `internal/symbols/extract_test.go`

**Step 1: Write the tests**

Create `internal/symbols/extract_test.go`:
```go
package symbols

import (
	"testing"
)

const goSrc = `package auth

type AuthService struct{}

func Login(user, pass string) bool {
	return validatePassword(user, pass)
}

func (s *AuthService) Logout(user string) {
	clearSession(user)
}
`

const tsSrc = `
class AuthService {
  login(user: string): boolean {
    return this.validatePassword(user);
  }
}

function logout(user: string): void {
  clearSession(user);
}
`

const pySrc = `
class AuthService:
    def login(self, user, password):
        return self.validate(user, password)

def logout(user):
    clear_session(user)
`

func TestExtractGo_Functions(t *testing.T) {
	nodes, _ := Extract("auth/service.go", "go", []byte(goSrc))
	names := nodeNames(nodes)

	assertContains(t, names, "Login")
	assertContains(t, names, "AuthService.Logout")
	assertContains(t, names, "AuthService")
}

func TestExtractGo_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.go", "go", []byte(goSrc))
	assertEdge(t, calls, "Login", "validatePassword")
	assertEdge(t, calls, "AuthService.Logout", "clearSession")
}

func TestExtractTS_ClassAndMethods(t *testing.T) {
	nodes, _ := Extract("auth/service.ts", "typescript", []byte(tsSrc))
	names := nodeNames(nodes)

	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_ClassAndMethods(t *testing.T) {
	nodes, _ := Extract("auth/service.py", "python", []byte(pySrc))
	names := nodeNames(nodes)

	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.py", "python", []byte(pySrc))
	assertEdge(t, calls, "AuthService.login", "self.validate")
	assertEdge(t, calls, "logout", "clear_session")
}

func TestExtract_UnknownLanguage_ReturnsNil(t *testing.T) {
	nodes, calls := Extract("file.rb", "ruby", []byte("def foo; end"))
	if nodes != nil || calls != nil {
		t.Error("expected nil for unknown language")
	}
}

// --- helpers ---

func nodeNames(nodes interface{ Names() []string }) []string {
	// We can't call methods on []CodeIndexNode directly, use inline helper below
	return nil
}

func assertContains(t *testing.T, names []string, want string) {
	t.Helper()
	for _, n := range names {
		if n == want {
			return
		}
	}
	t.Errorf("symbol %q not found in %v", want, names)
}

func assertEdge(t *testing.T, calls interface{}, from, to string) {
	t.Helper()
	// will be replaced with concrete type in actual implementation
}
```

Wait — the helper functions need concrete types. Rewrite with concrete `pkg` types:

```go
package symbols

import (
	"testing"

	"github.com/beakon/beakon/pkg"
)

const goSrc = `package auth

type AuthService struct{}

func Login(user, pass string) bool {
	return validatePassword(user, pass)
}

func (s *AuthService) Logout(user string) {
	clearSession(user)
}
`

const tsSrc = `
class AuthService {
  login(user) {
    return this.validatePassword(user);
  }
}

function logout(user) {
  clearSession(user);
}
`

const pySrc = `
class AuthService:
    def login(self, user, password):
        return self.validate(user, password)

def logout(user):
    clear_session(user)
`

func TestExtractGo_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.go", "go", []byte(goSrc))
	names := symbolNames(nodes)
	assertContains(t, names, "Login")
	assertContains(t, names, "AuthService.Logout")
	assertContains(t, names, "AuthService")
}

func TestExtractGo_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.go", "go", []byte(goSrc))
	assertEdge(t, calls, "Login", "validatePassword")
	assertEdge(t, calls, "AuthService.Logout", "clearSession")
}

func TestExtractGo_LineNumbers(t *testing.T) {
	nodes, _ := Extract("auth/service.go", "go", []byte(goSrc))
	for _, n := range nodes {
		if n.Name == "Login" {
			if n.StartLine < 1 {
				t.Errorf("Login StartLine = %d, want >= 1", n.StartLine)
			}
			if n.EndLine < n.StartLine {
				t.Errorf("Login EndLine %d < StartLine %d", n.EndLine, n.StartLine)
			}
		}
	}
}

func TestExtractTS_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.ts", "typescript", []byte(tsSrc))
	names := symbolNames(nodes)
	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.py", "python", []byte(pySrc))
	names := symbolNames(nodes)
	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.py", "python", []byte(pySrc))
	assertEdge(t, calls, "logout", "clear_session")
}

func TestExtract_UnknownLanguage(t *testing.T) {
	nodes, calls := Extract("file.rb", "ruby", []byte("def foo; end"))
	if len(nodes) != 0 || len(calls) != 0 {
		t.Error("expected empty results for unknown language")
	}
}

func symbolNames(nodes []pkg.CodeIndexNode) []string {
	var names []string
	for _, n := range nodes {
		names = append(names, n.Name)
	}
	return names
}

func assertContains(t *testing.T, names []string, want string) {
	t.Helper()
	for _, n := range names {
		if n == want {
			return
		}
	}
	t.Errorf("symbol %q not found in %v", want, names)
}

func assertEdge(t *testing.T, calls []pkg.CallEdge, from, to string) {
	t.Helper()
	for _, c := range calls {
		if c.From == from && c.To == to {
			return
		}
	}
	t.Errorf("call edge %q → %q not found", from, to)
}
```

**Step 2: Run tests**

```bash
go test ./internal/symbols/... -v
```
Expected: all tests PASS. If any fail, the parser is not extracting what the audit claimed — fix before continuing.

**Step 3: Commit**

```bash
git add internal/symbols/extract_test.go
git commit -m "test: symbol extraction tests for Go, TypeScript, Python"
```

---

### Task 10: Test suite — call graph

**Files:**
- Create: `internal/graph/build_test.go`

**Step 1: Write tests**

Create `internal/graph/build_test.go`:
```go
package graph

import (
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/pkg"
)

func edges(pairs ...string) []pkg.CallEdge {
	var out []pkg.CallEdge
	for i := 0; i < len(pairs)-1; i += 2 {
		out = append(out, pkg.CallEdge{From: pairs[i], To: pairs[i+1]})
	}
	return out
}

func TestBuild_DirectionsCorrect(t *testing.T) {
	from, to := Build(edges(
		"Login", "validatePassword",
		"Login", "createJWT",
		"Logout", "clearSession",
	))

	assertSliceContains(t, from["Login"], "validatePassword")
	assertSliceContains(t, from["Login"], "createJWT")
	assertSliceContains(t, from["Logout"], "clearSession")

	assertSliceContains(t, to["validatePassword"], "Login")
	assertSliceContains(t, to["createJWT"], "Login")
	assertSliceContains(t, to["clearSession"], "Logout")
}

func TestBuild_Deduplicates(t *testing.T) {
	from, _ := Build(edges(
		"Login", "validatePassword",
		"Login", "validatePassword", // duplicate
	))
	if len(from["Login"]) != 1 {
		t.Errorf("expected 1 unique callee, got %d", len(from["Login"]))
	}
}

func TestTrace_BFS(t *testing.T) {
	from, _ := Build(edges(
		"Login", "validatePassword",
		"validatePassword", "hashPassword",
	))
	result := Trace("Login", from)
	assertSliceContains(t, result, "Login")
	assertSliceContains(t, result, "validatePassword")
	assertSliceContains(t, result, "hashPassword")
}

func TestTrace_CycleDetection(t *testing.T) {
	from, _ := Build(edges(
		"A", "B",
		"B", "A", // cycle
	))
	result := Trace("A", from)
	// Should not loop forever; should have exactly 2 unique symbols
	if len(result) != 2 {
		t.Errorf("expected 2 results with cycle, got %d: %v", len(result), result)
	}
}

func TestWriteRead_RoundTrip(t *testing.T) {
	root := t.TempDir()
	from, to := Build(edges("Login", "validatePassword"))

	if err := Write(root, from, to); err != nil {
		t.Fatal(err)
	}

	gotFrom, err := ReadFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	gotTo, err := ReadTo(root)
	if err != nil {
		t.Fatal(err)
	}

	assertSliceContains(t, gotFrom["Login"], "validatePassword")
	assertSliceContains(t, gotTo["validatePassword"], "Login")
}

func TestWrite_NoTmpFilesLeft(t *testing.T) {
	root := t.TempDir()
	from, to := Build(edges("A", "B"))
	if err := Write(root, from, to); err != nil {
		t.Fatal(err)
	}
	// Verify no .tmp files in graph dir
	entries, _ := filepath.Glob(filepath.Join(root, graphDir, "*.tmp"))
	if len(entries) > 0 {
		t.Errorf("tmp files left after Write: %v", entries)
	}
}

func assertSliceContains(t *testing.T, slice []string, want string) {
	t.Helper()
	for _, s := range slice {
		if s == want {
			return
		}
	}
	t.Errorf("%q not found in %v", want, slice)
}
```

**Step 2: Run**

```bash
go test ./internal/graph/... -v
```
Expected: all PASS.

**Step 3: Commit**

```bash
git add internal/graph/build_test.go
git commit -m "test: call graph build, trace, BFS, cycle detection, write/read"
```

---

### Task 11: Test suite — index storage

**Files:**
- Modify: `internal/index/write_test.go` (already created in Task 5 — extend it)

**Step 1: Add more tests to write_test.go**

Append to `internal/index/write_test.go`:
```go
func TestReadAll_LoadsAllFiles(t *testing.T) {
	root := t.TempDir()
	Init(root)

	files := []pkg.FileIndex{
		{File: "auth/service.go", Hash: "hash1", Symbols: []pkg.CodeIndexNode{{Name: "Login"}}},
		{File: "api/controller.go", Hash: "hash2", Symbols: []pkg.CodeIndexNode{{Name: "Handle"}}},
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
	// File not indexed yet — should need update
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
	syms := []pkg.CodeIndexNode{{Name: "Login"}, {Name: "Logout"}}
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
```

**Step 2: Run**

```bash
go test ./internal/index/... -v
```
Expected: all PASS.

**Step 3: Commit**

```bash
git add internal/index/write_test.go
git commit -m "test: index storage write, read, delete, hash comparison"
```

---

### Task 12: Test suite — integration (end-to-end)

This verifies the full pipeline using the existing `testdata/sample_repo`.

**Files:**
- Create: `internal/indexer/integration_test.go`

**Step 1: Write integration test**

```go
package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/internal/context"
	"github.com/beakon/beakon/internal/index"
)

func TestFullPipeline_IndexAndQuery(t *testing.T) {
	// Copy testdata/sample_repo to a temp dir so we don't pollute it
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
	if result.Symbols == 0 {
		t.Error("expected at least 1 symbol indexed")
	}

	// 2. Symbols persisted
	syms, err := index.ReadSymbols(root)
	if err != nil {
		t.Fatalf("ReadSymbols: %v", err)
	}
	assertSymbolExists(t, syms, "AuthService.Login")

	// 3. Context engine returns a bundle
	engine := context.NewEngine(root)
	bundle, err := engine.Assemble("AuthService.Login")
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if bundle.Anchor.Name != "AuthService.Login" {
		t.Errorf("anchor = %q, want AuthService.Login", bundle.Anchor.Name)
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

func assertSymbolExists(t *testing.T, syms interface{ Names() []string }, name string) {
	t.Helper()
	// inline since []pkg.CodeIndexNode doesn't have a Names() method
}

// rewrite with concrete type:

func TestFullPipeline_IndexAndQuery2(t *testing.T) {
	root := t.TempDir()
	copyDir(t, "../../testdata/sample_repo", root)

	result, err := Run(root)
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("index failed: %v %v", err, result.Errors)
	}

	syms, _ := index.ReadSymbols(root)
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
```

**Step 2: Run**

```bash
go test ./internal/indexer/... -v -run TestFullPipeline
```
Expected: PASS. This exercises scan → parse → index → graph → context end to end.

**Step 3: Run all tests**

```bash
go test ./...
```
Expected: all PASS, no failures.

**Step 4: Commit**

```bash
git add internal/indexer/integration_test.go
git commit -m "test: end-to-end integration test covering index, update, and context"
```

---

### Task 13: Implement `impact` command

**What it does:** Given a symbol, return every symbol that would be affected if it changed — i.e., reverse BFS through `calls_to`.

Example:
```bash
./beakon impact validatePassword
# → AuthService.Login depends on validatePassword
# → UserController.Login depends on AuthService.Login
```

**Files:**
- Modify: `internal/graph/build.go` — add `Impact()` function
- Modify: `internal/graph/build_test.go` — test it
- Modify: `cmd/beakon/main.go` — add `impactCmd`

**Step 1: Write the failing test for Impact**

Add to `internal/graph/build_test.go`:
```go
func TestImpact_ReverseBFS(t *testing.T) {
	_, to := Build(edges(
		"Login", "validatePassword",
		"Login", "createJWT",
		"Handle", "Login",
	))

	result := Impact("validatePassword", to)

	assertSliceContains(t, result, "validatePassword")
	assertSliceContains(t, result, "Login")
	assertSliceContains(t, result, "Handle")
}

func TestImpact_CycleDetection(t *testing.T) {
	_, to := Build(edges(
		"A", "B",
		"B", "A",
	))
	result := Impact("A", to)
	if len(result) > 2 {
		t.Errorf("cycle not detected: got %d results", len(result))
	}
}

func TestImpact_IsolatedSymbol(t *testing.T) {
	_, to := Build(edges("A", "B"))
	result := Impact("A", to) // nothing calls A
	if len(result) != 1 {
		t.Errorf("expected only anchor symbol, got %v", result)
	}
}
```

**Step 2: Run to confirm failure**

```bash
go test ./internal/graph/... -v -run TestImpact
```
Expected: FAIL — `Impact` undefined.

**Step 3: Implement Impact in internal/graph/build.go**

Add after the `Trace` function (around line 94):
```go
// Impact performs reverse BFS from a symbol through calls_to.
// It returns every symbol that directly or indirectly depends on the given symbol.
// This is the answer to: "if I change X, what else might break?"
func Impact(symbol string, to CallsTo) []string {
	queue := []string{symbol}
	visited := map[string]bool{}
	var result []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)
		for _, caller := range to[current] {
			queue = append(queue, caller)
		}
	}
	return result
}
```

**Step 4: Run tests to confirm pass**

```bash
go test ./internal/graph/... -v -run TestImpact
```
Expected: all PASS.

**Step 5: Add `impact` command to cmd/beakon/main.go**

Find the `var rootCmd` block and add `impactCmd` alongside the other commands. Follow the exact same pattern as `callersCmd`:

```go
var impactCmd = &cobra.Command{
	Use:   "impact <symbol>",
	Short: "Show every symbol that depends on this one (reverse impact analysis)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		human, _ := cmd.Flags().GetBool("human")
		root, _ := os.Getwd()
		sym := args[0]

		callsTo, err := graph.ReadTo(root)
		if err != nil {
			return fmt.Errorf("load graph: %w", err)
		}

		result := graph.Impact(sym, callsTo)

		if human {
			fmt.Printf("Impact of changing %q (%d affected symbols):\n\n", sym, len(result))
			for i, s := range result {
				if i == 0 {
					fmt.Printf("  [anchor] %s\n", s)
				} else {
					fmt.Printf("  → %s\n", s)
				}
			}
		} else {
			printJSON(map[string]any{
				"symbol":  sym,
				"impact":  result,
				"count":   len(result),
			})
		}
		return nil
	},
}
```

Register it in `init()` or wherever other commands are added:
```go
rootCmd.AddCommand(impactCmd)
impactCmd.Flags().Bool("human", false, "human-readable output")
```

**Step 6: Build and smoke test**

```bash
go build -o beakon ./cmd/beakon
./beakon index
./beakon impact validatePassword --human
```
Expected output similar to:
```
Impact of changing "validatePassword" (2 affected symbols):

  [anchor] validatePassword
  → AuthService.Login
```

**Step 7: Commit**

```bash
git add internal/graph/build.go internal/graph/build_test.go cmd/beakon/main.go
git commit -m "feat: implement impact command with reverse BFS through calls_to graph"
```

---

### Task 14: .gitignore integration in repo scanner

**Problem:** The scanner ignores a hardcoded list of directories but does not read `.gitignore`. Real projects have custom ignore rules; without this, Beakon indexes generated files, build artifacts, and vendored copies.

**Files:**
- Modify: `internal/repo/scan.go`

**Step 1: Write the failing test**

Create `internal/repo/scan_test.go`:
```go
package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_RespectsGitignore(t *testing.T) {
	root := t.TempDir()

	// Create source files
	os.MkdirAll(filepath.Join(root, "src"), 0755)
	os.MkdirAll(filepath.Join(root, "generated"), 0755)
	os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(root, "generated", "types.go"), []byte("package gen"), 0644)

	// Write .gitignore that excludes generated/
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("generated/\n"), 0644)

	files, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if contains(f.Path, "generated") {
			t.Errorf("file in ignored dir included: %s", f.Path)
		}
	}

	found := false
	for _, f := range files {
		if contains(f.Path, "main.go") {
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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to confirm failure**

```bash
go test ./internal/repo/... -v -run TestScan_RespectsGitignore
```
Expected: FAIL — `generated/types.go` is included despite `.gitignore`.

**Step 3: Implement .gitignore parsing in scan.go**

Read the current `scan.go` first. It uses `filepath.WalkDir`. Add a `.gitignore` loader:

At the top of `scan.go`, add a helper that reads `.gitignore` from the root and returns a set of ignored patterns:

```go
// loadGitignore reads root/.gitignore and returns a set of directory/file patterns to skip.
// Only simple patterns are supported (no glob negation, no complex rules).
func loadGitignore(root string) map[string]bool {
	ignored := make(map[string]bool)
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return ignored // no .gitignore — that's fine
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Normalize: strip trailing slash (we match dir names)
		line = strings.TrimSuffix(line, "/")
		ignored[line] = true
	}
	return ignored
}
```

In `Scan()`, call `loadGitignore(root)` before the walk, then check each directory name against the ignore set inside the walk callback — the same way the existing `skipDirs` set is checked.

**Step 4: Run test**

```bash
go test ./internal/repo/... -v
```
Expected: all PASS.

**Step 5: Build**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add internal/repo/scan.go internal/repo/scan_test.go
git commit -m "feat: respect .gitignore patterns during repository scan"
```

---

### Task 15: Qualify call edges by receiver/package (Missing Capability — cross-file type resolution)

**Problem:** Call edges store raw string names. `"validatePassword"` could exist in multiple files/types. The graph connects all of them as the same symbol, producing false edges in any real-world repo. The fix: during extraction, propagate the enclosing type/package context into call edge targets so they become qualified names.

This is a partial fix — full type inference is out of scope — but qualifying calls made via `self`/`this`/receiver to the enclosing type resolves the most common ambiguity.

**Files:**
- Modify: `internal/symbols/extract.go` — `goCallEdges`, `tsCallEdges`, `pyCallEdges`
- Modify: `internal/symbols/extract_test.go` — tests to verify qualified edges

**Step 1: Write failing tests**

Add to `internal/symbols/extract_test.go`:
```go
const goMethodCallSrc = `package auth

type AuthService struct{}

func (s *AuthService) Login(user string) bool {
	return s.validatePassword(user)
}

func (s *AuthService) validatePassword(user string) bool {
	return true
}
`

func TestExtractGo_MethodCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.go", "go", []byte(goMethodCallSrc))
	// s.validatePassword inside AuthService.Login should resolve to AuthService.validatePassword
	assertEdge(t, calls, "AuthService.Login", "AuthService.validatePassword")
}

const tsMethodCallSrc = `
class AuthService {
  login(user) {
    return this.validatePassword(user);
  }
  validatePassword(user) { return true; }
}
`

func TestExtractTS_ThisCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.ts", "typescript", []byte(tsMethodCallSrc))
	// this.validatePassword inside AuthService.login should resolve to AuthService.validatePassword
	assertEdge(t, calls, "AuthService.login", "AuthService.validatePassword")
}

const pyMethodCallSrc = `
class AuthService:
    def login(self, user):
        return self.validate(user)
    def validate(self, user):
        return True
`

func TestExtractPython_SelfCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.py", "python", []byte(pyMethodCallSrc))
	// self.validate inside AuthService.login should resolve to AuthService.validate
	assertEdge(t, calls, "AuthService.login", "AuthService.validate")
}
```

**Step 2: Run to confirm failure**

```bash
go test ./internal/symbols/... -v -run TestExtract.*Qualified
```
Expected: FAIL — edges currently use unqualified callee names.

**Step 3: Update goCallEdges to qualify receiver calls**

In `internal/symbols/extract.go`, update `goCallEdges` to accept the enclosing receiver type name and qualify `receiver.method` calls:

```go
func goCallEdges(n *sitter.Node, src []byte, from string, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					// Qualify "s.Method" or "self.Method" as "ReceiverType.Method"
					callee = qualifyCall(callee, receiver)
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}

// qualifyCall rewrites "s.Foo" or "self.Foo" to "ReceiverType.Foo" when receiverType is known.
// e.g. qualifyCall("s.validatePassword", "AuthService") → "AuthService.validatePassword"
func qualifyCall(callee, receiverType string) string {
	if receiverType == "" {
		return callee
	}
	// Match single-identifier prefix followed by dot: "s.Foo", "self.Foo", "r.Foo"
	dot := strings.Index(callee, ".")
	if dot <= 0 {
		return callee
	}
	// Only qualify if it looks like a receiver variable (lowercase, short)
	prefix := callee[:dot]
	method := callee[dot+1:]
	// Heuristic: short prefix (1-6 chars) that is lowercase → likely a receiver/self variable
	if len(prefix) <= 6 && len(prefix) > 0 && prefix[0] >= 'a' && prefix[0] <= 'z' {
		return receiverType + "." + method
	}
	return callee
}
```

Pass `receiver` into `goCallEdges` at the call sites in `extractGo`:
- For `function_declaration`: pass `""` (no receiver)
- For `method_declaration`: pass the `receiver` string already extracted

**Step 4: Update tsCallEdges to qualify this. calls**

In `tsCallEdges`, add `parent string` parameter and qualify `this.X` calls:

```go
func tsCallEdges(n *sitter.Node, src []byte, from string, parent string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					// "this.validatePassword" → "AuthService.validatePassword"
					if parent != "" && strings.HasPrefix(callee, "this.") {
						callee = parent + "." + callee[5:]
					}
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}
```

Update call sites in `extractTS` to pass `parent`.

**Step 5: Update pyCallEdges to qualify self. calls**

```go
func pyCallEdges(n *sitter.Node, src []byte, from string, parent string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					// "self.validate" → "AuthService.validate"
					if parent != "" && strings.HasPrefix(callee, "self.") {
						callee = parent + "." + callee[5:]
					}
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}
```

Update call sites in `extractPython` to pass `parent`.

**Step 6: Run all symbol tests**

```bash
go test ./internal/symbols/... -v
```
Expected: all PASS including the new Qualified tests.

**Step 7: Run full test suite**

```bash
go test ./...
```
Expected: all PASS. The graph and context tests still pass because the graph is just string maps — it doesn't care whether edge names are qualified or not.

**Step 8: Commit**

```bash
git add internal/symbols/extract.go internal/symbols/extract_test.go
git commit -m "feat: qualify call edges for self./this./receiver. method calls"
```

---

### Task 16: Progress output during indexing (Missing Capability — UX)

**Problem:** `./beakon index` runs silently until completion. On a repo with 500+ files this looks like a hang. Users (and AI agents in verbose mode) need incremental feedback.

**Files:**
- Modify: `internal/indexer/index_repo.go` — add progress callback
- Modify: `cmd/beakon/main.go` — wire callback to stderr output

**Step 1: Add a Progress callback to the indexer**

In `internal/indexer/index_repo.go`, add a `ProgressFunc` type and a field to pass it into `Run`:

```go
// ProgressFunc is called each time a file is processed.
// indexed = files newly indexed, skipped = files skipped, total = total files.
type ProgressFunc func(indexed, skipped, total int)
```

Add it as an optional parameter by changing `Run`'s signature to accept options:

```go
// RunOptions configures an indexing run.
type RunOptions struct {
	// Progress is called after each file is processed. May be nil.
	Progress ProgressFunc
}

// Run performs a full index of the repository at root.
func Run(root string, opts ...RunOptions) (*Result, error) {
	var opt RunOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	// ... existing body ...
```

Inside the worker goroutine, after incrementing `indexed` or `skipped`, call the progress function if set:

```go
mu.Lock()
allSymbols = append(allSymbols, syms...)
allEdges = append(allEdges, calls...)
indexed++
if opt.Progress != nil {
    opt.Progress(indexed, skipped, len(files))
}
mu.Unlock()
```

Do the same in the skipped-file path.

**Step 2: Wire progress to stderr in the index command**

In `cmd/beakon/main.go`, find the `indexCmd` RunE function. Add `--progress` flag (enabled by default for human output). Replace the `Run(root)` call with:

```go
var lastPct int
progressFn := func(indexed, skipped, total int) {
    done := indexed + skipped
    pct := done * 100 / total
    // Only print at 10% increments to avoid spamming
    if pct >= lastPct+10 || done == total {
        lastPct = pct
        fmt.Fprintf(os.Stderr, "\rindexing... %d/%d files (%d%%)", done, total, pct)
        if done == total {
            fmt.Fprintln(os.Stderr)
        }
    }
}

var runOpts indexer.RunOptions
if human {
    runOpts.Progress = progressFn
}
result, err := indexer.Run(root, runOpts)
```

**Step 3: Verify existing tests still compile**

All existing tests call `Run(root)` with no opts — the variadic signature makes this backwards compatible.

```bash
go test ./...
```
Expected: all PASS.

**Step 4: Build and test manually**

```bash
go build -o beakon ./cmd/beakon
./beakon index --human
```
Expected: progress line updates on stderr:
```
indexing... 1/2 files (50%)
indexing... 2/2 files (100%)
indexed 2 files, 8 symbols in 45ms
```

**Step 5: Commit**

```bash
git add internal/indexer/index_repo.go cmd/beakon/main.go
git commit -m "feat: progress output during indexing via optional ProgressFunc callback"
```

---

## FINAL VERIFICATION

**Step 1: Run all tests with race detector**

```bash
go test -race ./...
```
Expected: all PASS, no data races detected.

**Step 2: Full end-to-end smoke test**

```bash
go build -o beakon ./cmd/beakon
./beakon index --human
./beakon map --human
./beakon context AuthService.Login --human
./beakon impact validatePassword --human
./beakon callers Login --human
./beakon watch &
# make a code change, observe output
kill %1
```

**Step 3: Verify storage layout**

```bash
ls .beakon/
# meta.json  symbols.json  map.json  nodes/  graph/
ls .beakon/nodes/
# auth_service.go.json  api_controller.go.json
```

Expected: `.beakon/nodes/` exists (not `files/`), `.codeindex/` does not exist.

**Step 4: Tag release**

```bash
git tag v0.2.0-beakon
```

---

## SUMMARY OF CHANGES

| Task | Type | Category | Files |
|------|------|----------|-------|
| 1 | Rename | Module path | go.mod + all .go imports |
| 2 | Rename | Binary + dir | cmd/beakon/, CLAUDE.md |
| 3 | Rename | Storage dirs (.codeindex→.beakon, files→nodes) | write.go, build.go, scan.go, watch.go |
| 4 | Rename | Docs + error strings | all .md files, context/engine.go |
| 5 | Fix | Risk: atomic writes | index/write.go, graph/build.go |
| 6 | Fix | Risk: concurrent write safety | indexer/update.go |
| 7 | Fix | Risk: signal handling | cmd/beakon/main.go |
| 8 | Fix | Risk: panic recovery | indexer/index_repo.go |
| 9 | Test | Symbol extraction | symbols/extract_test.go |
| 10 | Test | Call graph | graph/build_test.go |
| 11 | Test | Index storage | index/write_test.go |
| 12 | Test | Integration (end-to-end) | indexer/integration_test.go |
| 13 | Feature | Missing: `impact` command | graph/build.go, main.go |
| 14 | Feature | Missing: .gitignore integration | repo/scan.go |
| 15 | Feature | Missing: qualify call edges | symbols/extract.go |
| 16 | Feature | Missing: indexing progress output | indexer/index_repo.go, main.go |

**NOT in this plan (premature):** LSP server, distributed indexing, vector embeddings, GUI, plugin system.
