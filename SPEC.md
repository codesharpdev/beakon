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
}
```

### FileIndex

Stored per source file in .beakon/files/*.json

```go
type FileIndex struct {
    File    string          `json:"file"`
    Hash    string          `json:"hash"`
    Symbols []BeakonNode `json:"symbols"`
    Calls   []CallEdge      `json:"calls"`
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
├── files/
│   └── *.json          one FileIndex per source file
└── graph/
    ├── calls_from.json  symbol → []symbols it calls
    └── calls_to.json    symbol → []symbols that call it
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

### files/auth_service.go.json

```json
{
  "file": "auth/service.go",
  "hash": "a1b2c3d4e5f6",
  "symbols": [...],
  "calls": [...]
}
```

Note: file path slashes are replaced with underscores for the filename.
auth/service.go → auth_service.go.json

---

## Supported Languages

Phase 1:

    go
    typescript
    javascript
    python

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

## Performance Targets

| Operation              | Target  |
|------------------------|---------|
| Any query command      | <100ms  |
| Incremental file update| <200ms  |
| Full index (medium)    | <30s    |
