# Beakon — Repository Rules

These rules are invariants. They must not be broken.
If a change requires breaking a rule, discuss it first by updating this file.

---

## Package Structure Rules

Allowed top-level directories:

    cmd/
    internal/
    pkg/
    testdata/
    docs/      (optional)

Forbidden:

    scripts/
    misc/
    tmp/
    utils/

All core logic lives in internal/.
All shared types live in pkg/.
cmd/ contains only CLI wiring — no business logic.

---

## Import Rules

Packages may only import at the same level or below.

Allowed:
    cmd/beakon       → internal/context, internal/indexer, internal/graph,
                       internal/index, internal/code, pkg
    internal/context  → internal/graph, internal/index, internal/code, pkg
    internal/indexer  → internal/repo, internal/symbols, internal/resolver,
                        internal/graph, internal/index, pkg
    internal/resolver → pkg (+ stdlib only for file I/O)
    internal/graph    → pkg
    internal/index    → pkg
    internal/symbols  → pkg
    internal/repo     → stdlib only
    internal/code     → stdlib only
    pkg               → stdlib only

Forbidden:
    internal/* → cmd/*
    internal/graph → internal/indexer
    internal/index → internal/graph
    internal/context → internal/indexer
    internal/symbols → internal/resolver
    pkg → internal/*

Circular imports are forbidden.

---

## Data Model Rules

All symbols must be represented by pkg.BeakonNode.

Do NOT create alternative node structs.
Do NOT add fields to BeakonNode or CallEdge without updating SPEC.md.

Schema changes require:
1. Update pkg/types.go
2. Update SPEC.md
3. Update all code that reads/writes the struct
4. Migration note in TASKS.md

---

## Node ID Rules

Node IDs must follow exactly:

    <language>:<kind>:<filepath>:<symbol>

Example:

    go:method:auth/service.go:AuthService.Login

IDs must be:
- Deterministic (same inputs → same ID always)
- Stable (does not change unless file path or name changes)
- Globally unique within a repository

Random IDs, UUIDs, and timestamp-based IDs are forbidden.

---

## Storage Rules

All index data lives inside .beakon/ only.
Format: JSON always.
Binary formats: forbidden.
Database files: forbidden.
No SQLite, BoltDB, or any embedded database.

.beakon/ is gitignored (derived artifact).

Writes to symbols.json, map.json, calls_from.json, calls_to.json, external.json
must be atomic: write to temp file → os.Rename to final path.

---

## Parser Rules

Parsing must use Tree-sitter via github.com/smacker/go-tree-sitter.
Custom parsers are forbidden.
Regex-based symbol extraction is forbidden.

If a language is not supported by go-tree-sitter, it is not supported by Beakon.

Tree-sitter parsers are not thread-safe. Create a new parser per goroutine.

---

## Resolver Rules

Resolver (internal/resolver) may only read source files and lockfiles.
It must not call the network.
It must not execute any code.
It must not write to the index.

Enrichment must be a pure annotation pass on []CallEdge.

---

## Performance Rules

These are hard limits, not guidelines:

    Query response:           <100ms
    Incremental file update:  <200ms
    Full index (medium repo): <30s

Any PR that causes a benchmark to exceed these limits must be rejected
or must include a justification and updated benchmark baseline.

---

## Error Handling Rules

A single file parse failure must never abort a full index run.
Log the error, skip the file, continue.

A query against a missing index must print a clear message:
    "run 'beakon index' first"

Panics are forbidden in production code paths.
Use recover() in goroutines that parse untrusted source files.

---

## Test Rules

Every new internal/ package must have a _test.go file.
Tests must use testdata/sample_repo as the standard fixture.
Tests must not write to the real filesystem — use t.TempDir().
Benchmarks must be in _test.go files with Benchmark prefix.

Run before every commit:

    go test ./...
    go vet ./...

---

## Commit Rules

Format:

    <type>: <short description>

Types:
    feat     new feature
    fix      bug fix
    perf     performance improvement
    test     tests only
    docs     documentation only
    refactor code change with no behavior change

Examples:

    feat: add impact command
    fix: handle deleted files in watch mode
    perf: atomic writes for symbols.json
    test: add graph BFS cycle detection test
    feat: add resolver lockfile version pinning

---

## AI Agent Rules

Agents working in this repo must:

1. Read SPEC.md before touching data structures
2. Read ARCHITECTURE.md before touching package layout
3. Read TASKS.md before starting any implementation
4. Run go test ./... before reporting a task complete
5. Never introduce a package that violates the import rules
6. Never use grep/find/cat to explore — use beakon commands

Agents must not:
- Introduce global mutable state
- Add network calls
- Execute repository source code
- Modify source files (read-only analysis only)
