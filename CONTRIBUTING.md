# Beakon — Contributing

## Setup

Requirements: Go 1.21+

```bash
git clone https://github.com/codesharpdev/beakon
cd beakon
go mod tidy
go build -o beakon ./cmd/beakon
```

Verify:

```bash
./beakon index
./beakon map --human
./beakon context "AuthService.Login" --human
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

1. Add handler in cmd/beakon/main.go
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
6. Add builtin set in internal/resolver/builtins.go
7. Add import parsing in internal/resolver/imports.go
8. Add test cases in internal/symbols/extract_test.go
9. Update SPEC.md supported languages table

---

## Adding a New Lockfile Format

1. Add parser in internal/resolver/lockfile.go
2. Detect the file in ReadLockfile() by filename pattern
3. Extract: package name → version + devOnly bool
4. Add test cases in internal/resolver/resolver_test.go

---

## Modifying the Data Model

Before changing BeakonNode or CallEdge:
1. Read SPEC.md — understand current schema
2. Update pkg/types.go
3. Update SPEC.md to match
4. Find all callers of the struct — update them all
5. Check that existing .beakon/ files still load (or add migration)
6. Update TASKS.md with migration note

---

## Directory Reference

```
beakon/
├── CLAUDE.md        Agent operating instructions
├── SPEC.md          Data structures + storage layout
├── ARCHITECTURE.md  Package responsibilities + pipeline
├── TASKS.md         What is done, what is next
├── REPO_RULES.md    Hard invariants
├── TESTING.md       How to test everything
├── CONTRIBUTING.md  This file
│
├── cmd/beakon/main.go     CLI entry point (11 commands)
│
├── internal/
│   ├── repo/       Repository scanner (.gitignore-aware)
│   ├── symbols/    Tree-sitter parsing (18 languages)
│   ├── resolver/   External dependency enrichment
│   ├── graph/      Bidirectional call graph
│   ├── index/      Storage layer (.beakon/ JSON)
│   ├── indexer/    Orchestration (full + incremental + watch)
│   ├── context/    LLM context bundle assembly
│   └── code/       Live source extraction
│
├── pkg/types.go    Shared data structures
│
└── testdata/
    └── sample_repo/
        ├── auth/service.go      AuthService type + methods
        └── api/controller.go    UserController type + methods
```
