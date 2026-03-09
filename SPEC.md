# Beakon — Specification

## Core Data Structures

### BeakonNode

Defined in pkg/types.go.

```go
type BeakonNode struct {
    ID         string `json:"id"`
    Kind       string `json:"kind"`
    Name       string `json:"name"`
    Language   string `json:"language"`
    FilePath   string `json:"file_path"`
    StartLine  int    `json:"start_line"`
    EndLine    int    `json:"end_line"`
    Parent     string `json:"parent,omitempty"`
    SourceHash string `json:"source_hash"`
}
```

Allowed Kind values:

    function
    method
    class
    module
    variable

### CallEdge

```go
type CallEdge struct {
    From string `json:"from"`
    To   string `json:"to"`

    // External dependency enrichment (populated by internal/resolver)
    Package    string  `json:"package,omitempty"`
    Stdlib     string  `json:"stdlib,omitempty"`     // "yes" | "no" | "unknown"
    DevOnly    *bool   `json:"dev_only,omitempty"`   // nil = unknown
    Version    string  `json:"version,omitempty"`
    Resolution string  `json:"resolution,omitempty"` // "resolved" | "unresolved"
    Reason     string  `json:"reason,omitempty"`     // "dot_import" | "wildcard_import" | "no_import_found"
    Hint       string  `json:"hint,omitempty"`
}
```

Internal call edges (within the repo) have only From and To set.
External call edges additionally carry Package, Stdlib, Version, DevOnly, Resolution.

### ExternalCallee

```go
type ExternalCallee struct {
    Package    string `json:"package"`
    Stdlib     string `json:"stdlib"`
    Version    string `json:"version,omitempty"`
    DevOnly    *bool  `json:"dev_only,omitempty"`
    Resolution string `json:"resolution"`
    Reason     string `json:"reason,omitempty"`
    Hint       string `json:"hint,omitempty"`
}
```

### ExternalIndex

```go
type ExternalIndex map[string]ExternalCallee  // callee name → enrichment
```

Stored in .beakon/graph/external.json.

### FileIndex

Stored per source file in .beakon/nodes/*.json

```go
type FileIndex struct {
    File    string       `json:"file"`
    Hash    string       `json:"hash"`
    Symbols []BeakonNode `json:"symbols"`
    Calls   []CallEdge   `json:"calls"`
}
```

### TraceStep

Used by trace and explain commands.

```go
type TraceStep struct {
    Symbol  string `json:"symbol"`
    File    string `json:"file,omitempty"`
    Line    int    `json:"line,omitempty"`
    EndLine int    `json:"end_line,omitempty"`
    Snippet string `json:"snippet,omitempty"`
    Depth   int    `json:"depth"`
}
```

---

## Node ID Format

Format:

    <language>:<kind>:<filepath>:<symbol>

Examples:

    go:function:auth/service.go:login
    go:method:auth/service.go:AuthService.Login
    ts:class:auth/AuthService.ts:AuthService
    py:function:payments/stripe.py:create_charge

Rules:

- Deterministic — same inputs always produce same ID
- Stable — does not change unless file path or symbol name changes
- Globally unique within a repository

---

## Storage Layout

```
.beakon/
├── meta.json           index metadata
├── symbols.json        flat list of all symbols
├── map.json            dir → []symbol names
├── nodes/
│   └── *.json          one FileIndex per source file
└── graph/
    ├── calls_from.json  symbol → []symbols it calls
    ├── calls_to.json    symbol → []symbols that call it
    └── external.json    external callee enrichment data
```

### meta.json

```json
{
  "version": "1",
  "indexed_at": "2025-01-01T00:00:00Z",
  "repo_root": "/path/to/repo",
  "file_count": 42,
  "sym_count": 380
}
```

### symbols.json

```json
{
  "symbols": [
    {
      "id": "go:method:auth/service.go:AuthService.Login",
      "kind": "method",
      "name": "AuthService.Login",
      "language": "go",
      "file_path": "auth/service.go",
      "start_line": 30,
      "end_line": 42,
      "parent": "AuthService",
      "source_hash": "a1b2c3d4"
    }
  ]
}
```

### map.json

```json
{
  "auth": ["AuthService.Login", "AuthService.Logout", "createJWT"],
  "api":  ["UserController.Login"]
}
```

### calls_from.json

```json
{
  "AuthService.Login": ["validatePassword", "createJWT"],
  "UserController.Login": ["AuthService.Login"]
}
```

### calls_to.json

```json
{
  "createJWT":           ["AuthService.Login"],
  "AuthService.Login":   ["UserController.Login"]
}
```

### external.json

```json
{
  "bcrypt.GenerateFromPassword": {
    "package": "golang.org/x/crypto/bcrypt",
    "stdlib": "no",
    "version": "v0.17.0",
    "dev_only": false,
    "resolution": "resolved"
  },
  "fmt.Errorf": {
    "package": "fmt",
    "stdlib": "yes",
    "resolution": "resolved"
  }
}
```

### nodes/auth_service_go.json

```json
{
  "file": "auth/service.go",
  "hash": "a1b2c3d4e5f6",
  "symbols": [...],
  "calls": [
    { "from": "AuthService.Login", "to": "validatePassword" },
    { "from": "AuthService.Login", "to": "bcrypt.GenerateFromPassword",
      "package": "golang.org/x/crypto/bcrypt", "stdlib": "no", "version": "v0.17.0" }
  ]
}
```

Note: file path separators and dots are replaced with underscores for the filename.
auth/service.go → auth_service_go.json

---

## Supported Languages

| Language | Extensions |
|----------|-----------|
| Go | `.go` |
| TypeScript | `.ts`, `.tsx` |
| JavaScript | `.js`, `.jsx` |
| Python | `.py` |
| Rust | `.rs` |
| Java | `.java` |
| C | `.c`, `.h` |
| C++ | `.cpp`, `.cc`, `.cxx`, `.hpp` |
| C# | `.cs` |
| Ruby | `.rb` |
| Kotlin | `.kt`, `.kts` |
| Swift | `.swift` |
| PHP | `.php` |
| Scala | `.scala` |
| Elixir | `.ex`, `.exs` |
| OCaml | `.ml`, `.mli` |
| Elm | `.elm` |
| Groovy | `.groovy` |

All parsing uses Tree-sitter via github.com/smacker/go-tree-sitter.

---

## File Hash Algorithm

SHA1 of raw file bytes, hex encoded.
Used for incremental indexing — skip files whose hash has not changed.

---

## Snippet Extraction

For trace --human and explain --human:

- Read source file live (not from index)
- Extract lines [StartLine, min(StartLine+5, EndLine)]
- Cap at 6 lines to keep output readable
- Never store snippets in index — always read live

---

## External Dependency Enrichment

Enrichment is performed by internal/resolver at index time.

Per call edge, resolver:
1. Checks if callee is a language builtin (drops it if so)
2. Parses all import statements from the source file
3. Resolves qualifier → package import path
4. Classifies as stdlib vs third-party
5. Reads lockfile to get pinned version
6. For TypeScript/JavaScript: checks devDependencies

Lockfile support:

| Language | Lockfile |
|----------|---------|
| Go | `go.mod` |
| JavaScript/TypeScript | `package.json`, `package-lock.json` |
| Python | `requirements.txt`, `poetry.lock` |
| Rust | `Cargo.toml` |
| Ruby | `Gemfile.lock` |

---

## Context Bundle

Assembled by internal/context.Engine.Assemble(query).

```go
type Bundle struct {
    Query          string      // original search term
    Anchor         CodeBlock   // the queried symbol + source
    Callees        []CodeBlock // what anchor calls (internal: source, external: enrichment)
    Callers        []CodeBlock // what calls anchor (source)
    Files          []string    // unique files involved
    TokenEstimate  int         // estimated LLM token count (chars / 4)
}

type CodeBlock struct {
    Symbol     string  // symbol name
    File       string  // source file path
    Start      int     // start line
    End        int     // end line
    Code       string  // source code (internal symbols)
    // External enrichment (populated when Code is empty):
    Package    string
    Stdlib     string
    Version    string
    DevOnly    *bool
    Resolution string
    Reason     string
    Hint       string
}
```

---

## Performance Targets

| Operation              | Target  |
|------------------------|---------|
| Any query command      | <100ms  |
| Incremental file update| <200ms  |
| Full index (medium)    | <30s    |
