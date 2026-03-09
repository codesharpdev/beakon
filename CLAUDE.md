# CodeIndex — Claude Code Instructions

## What Is This Repo

CodeIndex is a structural code intelligence CLI for AI agents.
It parses repositories with Tree-sitter, builds a call graph, and
assembles complete context bundles so AI agents reason about the right code.

Think of it as: LSP for AI agents.

---

## Navigation Model

Three tools. Each has a distinct purpose.

    grep        → find text you know exists
    cat         → read a file you already located
    codeindex   → navigate when you don't know where to look

They are not competing. They are layered.

---

## Correct Agent Workflow

WRONG:
    grep "login"
    open 20 files
    guess which one matters

CORRECT:
    codeindex context AuthService.login
    ↓ receive: anchor code + callers + callees
    ↓ now you know exactly which files matter
    cat auth/service.go        ← only if you need full file
    grep "session" auth/        ← only if you need text search

---

## The Context Command (Primary Tool)

```bash
./codeindex context <symbol> --human
```

Returns:
- The symbol's source code
- Everything it calls (with source)
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
| Complete LLM context bundle | `./codeindex context <symbol> --human`    |
| Repo structure overview     | `./codeindex map --human`                 |
| Search for a symbol         | `./codeindex search <text> --human`       |
| Show symbol source          | `./codeindex show <symbol> --human`       |
| Trace execution flow        | `./codeindex trace <symbol> --human`      |
| Explain a feature           | `./codeindex explain <symbol> --human`    |
| Who calls a function        | `./codeindex callers <symbol> --human`    |
| What a function depends on  | `./codeindex deps <symbol> --human`       |

---

## First Thing Every Session

```bash
go build -o codeindex ./cmd/codeindex
./codeindex index
./codeindex map --human
```

---

## Before Modifying Any Symbol

```bash
./codeindex context <symbol> --human
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
go build -o codeindex ./cmd/codeindex
go test ./...
./codeindex index ./testdata/sample_repo
./codeindex context AuthService.Login --human
```

---

## Performance Requirements

- Query response:           <100ms
- Incremental file update:  <200ms
- Full index (medium repo): <30s

---

## Invariants

1. All parsing uses Tree-sitter only
2. All symbols use CodeIndexNode — no alternative structs
3. Node IDs: <language>:<kind>:<filepath>:<symbol>
4. Disk storage: JSON only
5. No filesystem scanning during queries
6. internal/ packages must not import cmd/
