# Beakon — Claude Code Instructions

## What Is This Repo

Beakon is shared code intelligence infrastructure for AI agents.
It parses repositories with Tree-sitter, builds a bidirectional call graph,
enriches every external dependency with version and provenance metadata,
and assembles complete context bundles so AI agents reason about the right code.

Think of it as: LSP for AI agents.

---

## Navigation Model

Three tools. Each has a distinct purpose.

    grep        → find text you know exists
    cat         → read a file you already located
    beakon      → navigate when you don't know where to look

They are not competing. They are layered.

---

## Correct Agent Workflow

WRONG:
    grep "login"
    open 20 files
    guess which one matters

CORRECT:
    beakon context AuthService.login
    ↓ receive: anchor code + callers + callees (with external dependency metadata)
    ↓ now you know exactly which files matter
    cat auth/service.go        ← only if you need full file
    grep "session" auth/        ← only if you need text search

---

## The Context Command (Primary Tool)

```bash
./beakon context <symbol> --human
```

Returns:
- The symbol's source code (anchor)
- Everything it calls — internal symbols with source, external with Package/Stdlib/Version metadata
- Everything that calls it (with source)
- All files involved
- Token estimate

This is the single command that replaces:
    grep + open files + guess architecture

Use this first for any task involving a symbol.

---

## All Commands

| Goal                        | Command                                   |
|-----------------------------|-------------------------------------------|
| Complete LLM context bundle | `./beakon context <symbol> --human`       |
| Blast radius of a change    | `./beakon impact <symbol> --human`        |
| Repo structure overview     | `./beakon map --human`                    |
| Search for a symbol         | `./beakon search <text> --human`          |
| Show symbol source          | `./beakon show <symbol> --human`          |
| Trace execution flow        | `./beakon trace <symbol> --human`         |
| Explain a feature           | `./beakon explain <symbol> --human`       |
| Who calls a function        | `./beakon callers <symbol> --human`       |
| What a function depends on  | `./beakon deps <symbol> --human`          |

---

## First Thing Every Session

```bash
go build -o beakon ./cmd/beakon
./beakon index
./beakon map --human
```

---

## Before Modifying Any Symbol

```bash
./beakon context <symbol> --human
./beakon impact <symbol> --human
```

This shows you the full blast radius before you touch anything.

---

## Output Modes

| Flag      | Output   | Use for            |
|-----------|----------|--------------------|
| (default) | JSON     | Agent consumption  |
| --human   | Readable | Debugging, display |

---

## Key Docs

- SPEC.md          — data structures and storage layout
- ARCHITECTURE.md  — pipeline and package responsibilities
- TASKS.md         — what is done, what is next
- REPO_RULES.md    — invariants that must not be broken
- TESTING.md       — how to test everything

---

## Build and Test

```bash
go mod tidy
go build -o beakon ./cmd/beakon
go test ./...
./beakon index
./beakon context "AuthService.Login" --human
```

---

## Package Map

| Package              | Responsibility                                      |
|----------------------|-----------------------------------------------------|
| cmd/beakon           | CLI only (11 commands). No business logic.          |
| internal/context     | Assemble complete LLM context bundles               |
| internal/indexer     | Orchestrate: full index, incremental update, watch  |
| internal/resolver    | Enrich external calls: imports + lockfiles + stdlib |
| internal/graph       | Build + query bidirectional call graph              |
| internal/index       | Read/write .beakon/ JSON storage                    |
| internal/symbols     | Tree-sitter AST → symbols + call edges (18 langs)   |
| internal/repo        | Repository file discovery (.gitignore-aware)        |
| internal/code        | Live source code extraction                         |
| pkg                  | Shared types: BeakonNode, CallEdge, FileIndex       |

---

## Performance Requirements

- Query response:           <100ms
- Incremental file update:  <200ms
- Full index (medium repo): <30s

---

## Invariants

1. All parsing uses Tree-sitter only
2. All symbols use pkg.BeakonNode — no alternative structs
3. Node IDs: <language>:<kind>:<filepath>:<symbol>
4. Disk storage: JSON only, all writes atomic
5. No filesystem scanning during queries
6. internal/ packages must not import cmd/
7. internal/resolver must not call the network or execute code
8. Tree-sitter parsers are not thread-safe — create one per goroutine
