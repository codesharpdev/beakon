# Beakon — Testing Guide

## Quick Start

```bash
# Build
go build -o beakon ./cmd/beakon

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
```

---

## Manual Smoke Test

Run this after every significant change to verify end-to-end behavior.

```bash
# 1. Build
go build -o beakon ./cmd/beakon

# 2. Index the sample repo
./beakon index

# 3. Map — should show auth/ and api/ directories
./beakon map --human

# 4. Context — primary command: anchor + callers + callees
./beakon context "AuthService.Login" --human

# 5. Impact — what depends on createJWT
./beakon impact createJWT --human

# 6. Trace — should show Login → AuthService.Login call chain
./beakon trace Login --human

# 7. Explain — should show full feature flow + files
./beakon explain Login --human

# 8. Callers — should return UserController.Login
./beakon callers "AuthService.Login" --human

# 9. Deps — should return validatePassword, createJWT (+ any external calls)
./beakon deps "AuthService.Login" --human

# 10. Show — should print the Login function source
./beakon show "AuthService.Login" --human

# 11. Search — should return all login-related symbols
./beakon search login --human

# 12. JSON output check — all commands default to JSON
./beakon map
./beakon context "AuthService.Login"
./beakon trace Login
./beakon callers "AuthService.Login"
./beakon impact createJWT
```

---

## Expected Output Reference

### beakon map --human

```
auth/
  AuthService.Login
  AuthService.Logout
  validatePassword
  createJWT
  invalidateToken
api/
  UserController.Login
  UserController.Logout
```

### beakon context AuthService.Login --human

```
=== ANCHOR ===
AuthService.Login  auth/service.go:14-42
<source code>

=== CALLS ===
validatePassword  auth/service.go:28-35
<source code>

createJWT  auth/service.go:35-45
<source code>

=== CALLED BY ===
UserController.Login  api/controller.go:14-28
<source code>

files: auth/service.go, api/controller.go
tokens: ~420
```

### beakon impact createJWT --human

```
impact: createJWT
affected (1):
  AuthService.Login  auth/service.go:14
```

### beakon callers AuthService.Login --human

```
callers of AuthService.Login:
  UserController.Login
```

### beakon deps AuthService.Login --human

```
deps of AuthService.Login:
  → validatePassword
  → createJWT
```

---

## Watch Mode Test

```bash
# Terminal 1 — start watch mode
./beakon watch --human

# Terminal 2 — modify a file
echo "// comment" >> ./testdata/sample_repo/auth/service.go

# Terminal 1 should print:
# ↻ auth/service.go  5→5 symbols  12ms
```

---

## Incremental Update Test

```bash
# Index the sample repo
./beakon index

# Check symbol count
./beakon map --human

# Add a new function to auth/service.go
cat >> ./testdata/sample_repo/auth/service.go << 'EOF'

func refreshToken(token string) string {
    return token + ".refreshed"
}
EOF

# Re-index (should skip unchanged files)
./beakon index

# Verify new symbol appears
./beakon search refresh --human
# Expected: refreshToken  auth/service.go:XX
```

---

## External Enrichment Test

```bash
# After indexing, check external.json
cat .beakon/graph/external.json

# Should contain entries like:
# {
#   "fmt.Errorf": { "package": "fmt", "stdlib": "yes" },
#   "errors.New": { "package": "errors", "stdlib": "yes" }
# }

# Context output should include external callee metadata
./beakon context "AuthService.Login"
# JSON should show callees with "stdlib", "package" fields
```

---

## JSON Output Validation

All commands must produce valid JSON when --human is not set.

```bash
./beakon map                          | python3 -m json.tool
./beakon context "AuthService.Login"  | python3 -m json.tool
./beakon trace Login                  | python3 -m json.tool
./beakon callers "AuthService.Login"  | python3 -m json.tool
./beakon deps "AuthService.Login"     | python3 -m json.tool
./beakon show "AuthService.Login"     | python3 -m json.tool
./beakon search login                 | python3 -m json.tool
./beakon explain Login                | python3 -m json.tool
./beakon impact createJWT             | python3 -m json.tool
```

All must exit 0 with valid JSON.

---

## Unit Test Locations

| Package              | Test File                                  |
|----------------------|--------------------------------------------|
| internal/symbols     | internal/symbols/extract_test.go           |
| internal/graph       | internal/graph/build_test.go               |
| internal/index       | internal/index/write_test.go               |
| internal/indexer     | internal/indexer/update_test.go            |
| internal/indexer     | internal/indexer/integration_test.go       |
| internal/repo        | internal/repo/scan_test.go                 |
| internal/code        | internal/code/fetch_test.go                |
| internal/resolver    | internal/resolver/resolver_test.go         |
| internal/context     | internal/context/engine_test.go            |

---

## Writing Tests — Rules

1. Use t.TempDir() for all filesystem operations — never write to repo root
2. Use testdata/sample_repo as the fixture for integration tests
3. Each test must be independent — no shared state between tests
4. Table-driven tests preferred for parser and graph logic

Example test structure:

```go
func TestExtractGoSymbols(t *testing.T) {
    src := []byte(`
package auth

func Login(user string) string {
    return createJWT(user)
}
`)
    syms, calls := Extract("auth/service.go", "go", src)

    if len(syms) != 1 {
        t.Fatalf("expected 1 symbol, got %d", len(syms))
    }
    if syms[0].Name != "Login" {
        t.Errorf("expected Login, got %s", syms[0].Name)
    }
    if len(calls) != 1 || calls[0].To != "createJWT" {
        t.Errorf("expected call to createJWT")
    }
}
```

---

## Benchmark Structure

```go
func BenchmarkFullIndex(b *testing.B) {
    root := "../../testdata/sample_repo"
    for i := 0; i < b.N; i++ {
        dir := b.TempDir()
        // copy sample_repo to dir
        indexer.Run(dir)
    }
}

func BenchmarkIncrementalUpdate(b *testing.B) {
    // setup index once, then update single file in loop
}

func BenchmarkContextAssemble(b *testing.B) {
    // setup engine once, then call Assemble in loop
}

func BenchmarkTraceQuery(b *testing.B) {
    // setup index once, run trace in loop
}
```

---

## CI Checklist

Before marking any task DONE:

- [ ] go build ./... passes
- [ ] go test ./... passes
- [ ] go vet ./... passes
- [ ] Smoke test steps 1-12 pass
- [ ] JSON output is valid for all commands
- [ ] No new panics under test
