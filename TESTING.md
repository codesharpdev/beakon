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
./beakon index ./testdata/sample_repo

# 3. Map — should show auth/ and api/ directories
./beakon map --human

# 4. Trace — should show Login → AuthService.Login call chain
./beakon trace Login --human

# 5. Explain — should show full feature flow + files
./beakon explain Login --human

# 6. Callers — should return UserController.Login
./beakon callers "AuthService.Login" --human

# 7. Deps — should return validatePassword, createJWT
./beakon deps "AuthService.Login" --human

# 8. Show — should print the Login function source
./beakon show "AuthService.Login" --human

# 9. Search — should return all login-related symbols
./beakon search login --human

# 10. JSON output check — all commands default to JSON
./beakon map
./beakon trace Login
./beakon callers "AuthService.Login"
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

### beakon trace Login --human

```
Login
  → AuthService.Login
      auth/service.go:14

      func (s *AuthService) Login(username, password string) (string, error) {
          if err := validatePassword(username, password); err != nil {
          ...

    → validatePassword
        auth/service.go:28
        ...

    → createJWT
        auth/service.go:35
        ...
```

### beakon explain Login --human

```
Feature: Login

Flow:
  Login  (auth/service.go:14)
  validatePassword  (auth/service.go:28)
  createJWT  (auth/service.go:35)

Files involved:
  auth/service.go
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
./beakon index ./testdata/sample_repo

# Check symbol count
./beakon map --human

# Add a new function to auth/service.go
cat >> ./testdata/sample_repo/auth/service.go << 'EOF'

func refreshToken(token string) string {
    return token + ".refreshed"
}
EOF

# Re-index (should skip unchanged files)
./beakon index ./testdata/sample_repo

# Verify new symbol appears
./beakon search refresh --human
# Expected: refreshToken  auth/service.go:XX
```

---

## JSON Output Validation

All commands must produce valid JSON when --human is not set.

```bash
# Validate JSON output for each command
./beakon map | python3 -m json.tool
./beakon trace Login | python3 -m json.tool
./beakon callers "AuthService.Login" | python3 -m json.tool
./beakon deps "AuthService.Login" | python3 -m json.tool
./beakon show "AuthService.Login" | python3 -m json.tool
./beakon search login | python3 -m json.tool
./beakon explain Login | python3 -m json.tool
```

All must exit 0 with valid JSON.

---

## Unit Test Locations

| Package              | Test File                              |
|----------------------|----------------------------------------|
| internal/symbols     | internal/symbols/extract_test.go       |
| internal/graph       | internal/graph/build_test.go           |
| internal/index       | internal/index/write_test.go           |
| internal/indexer     | internal/indexer/update_test.go        |
| internal/indexer     | internal/indexer/integration_test.go   |
| internal/repo        | internal/repo/scan_test.go             |
| internal/code        | internal/code/fetch_test.go            |

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

func BenchmarkTraceQuery(b *testing.B) {
    // setup index once
    // b.ResetTimer()
    // run trace in loop
}
```

---

## CI Checklist

Before marking any task DONE:

- [ ] go build ./... passes
- [ ] go test ./... passes
- [ ] go vet ./... passes
- [ ] Smoke test steps 1-10 pass
- [ ] JSON output is valid for all commands
- [ ] No new panics under test
