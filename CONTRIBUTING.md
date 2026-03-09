# CodeIndex — Contributing

## Setup

Requirements: Go 1.21+

```bash
git clone <repo>
cd codeindex
go mod tidy
go build -o codeindex ./cmd/codeindex
```

Verify:

```bash
./codeindex index ./testdata/sample_repo
./codeindex map --human
```

---

## Development Workflow

1. Read TASKS.md — find the first TODO task
2. Read SPEC.md and ARCHITECTURE.md — understand the design
3. Read REPO_RULES.md — understand what must not be broken
4. Implement the task
5. Write tests (see TESTING.md)
6. Run go test ./... — must pass
7. Run smoke test from TESTING.md — must pass
8. Commit with correct message format

---

## Commit Format

    <type>: <short description>

Types: feat, fix, perf, test, docs, refactor

---

## Adding a New Command

1. Add handler in cmd/codeindex/main.go
2. Add to root.AddCommand() in init()
3. Follow the pattern: getwd → call internal package → format output
4. No business logic in main.go
5. Add to TESTING.md smoke test section
6. Update TASKS.md status

---

## Adding a New Language

1. Find tree-sitter grammar: github.com/smacker/go-tree-sitter
2. Add to go.mod
3. Add to treeSitterLang() in internal/symbols/parse.go
4. Add extraction logic in internal/symbols/extract.go
5. Add extension mapping in internal/repo/scan.go
6. Add test cases in internal/symbols/extract_test.go
7. Update SPEC.md supported languages section

---

## Modifying the Data Model

Before changing CodeIndexNode:
1. Read SPEC.md — understand current schema
2. Update pkg/types.go
3. Update SPEC.md to match
4. Find all callers of the struct — update them all
5. Check that existing .codeindex/ files still load (or add migration)
6. Update TASKS.md with migration note

---

## Directory Reference

```
codeindex/
├── CLAUDE.md        Agent operating instructions
├── SPEC.md          Data structures + storage layout
├── ARCHITECTURE.md  Package responsibilities + pipeline
├── TASKS.md         What is done, what is next
├── REPO_RULES.md    Hard invariants
├── TESTING.md       How to test everything
├── CONTRIBUTING.md  This file
│
├── cmd/codeindex/main.go     CLI entry point
│
├── internal/
│   ├── repo/       Repository scanner
│   ├── symbols/    Tree-sitter parsing
│   ├── graph/      Call graph
│   ├── index/      Storage layer
│   ├── indexer/    Orchestration (full + incremental + watch)
│   └── code/       Live source extraction
│
├── pkg/types.go    Shared data structures
│
└── testdata/
    └── sample_repo/
        ├── auth/service.go
        └── api/controller.go
```
