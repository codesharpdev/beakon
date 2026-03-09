# Beakon

**Structural code intelligence CLI for AI agents.**

Beakon parses your repository with Tree-sitter, builds a bidirectional call graph, and assembles complete context bundles so AI agents reason about the right code — not random grep results.

Think of it as: **LSP for AI agents.**

---

## Why

AI agents navigating a codebase face a core problem: they don't know what they don't know.

The typical agent workflow:
```
grep "login" → open 20 files → guess which one matters
```

This wastes tokens on irrelevant code and misses the files that actually matter.

Beakon solves this with a single command:
```bash
beakon context AuthService.Login
```

You get the symbol's source, everything it calls, everything that calls it, all files involved, and a token estimate. The agent knows exactly where to look before touching anything.

---

## Installation

Requirements: **Go 1.21+**

```bash
git clone https://github.com/beakon/beakon
cd beakon
go build -o beakon ./cmd/beakon
```

Add to your PATH or use `./beakon` from the repo root.

---

## Quick Start

```bash
# Index your repository
beakon index

# Get complete context for a symbol
beakon context AuthService.Login --human

# See the repo structure
beakon map --human
```

---

## Commands

| Goal | Command |
|------|---------|
| Index the repository | `beakon index` |
| Watch for changes | `beakon watch` |
| **Complete LLM context bundle** | `beakon context <symbol>` |
| Repo structure overview | `beakon map` |
| Search for a symbol | `beakon search <text>` |
| Show symbol source | `beakon show <symbol>` |
| Trace execution flow | `beakon trace <symbol>` |
| Explain a feature | `beakon explain <symbol>` |
| Who calls a function | `beakon callers <symbol>` |
| What a function calls | `beakon deps <symbol>` |

All commands output **JSON by default** (for agent consumption) and readable text with `--human`.

---

## The `context` Command

The primary command. Assembles everything an LLM needs to reason about a symbol.

```bash
beakon context AuthService.Login --human
```

Returns:
- The symbol's full source code (anchor)
- Source of every function it calls (callees)
- Source of every function that calls it (callers)
- List of all files involved
- Token estimate

This is the single command that replaces `grep + open files + guess architecture`.

---

## Agent Workflow

**Wrong:**
```
grep "login"
open 20 files
guess which one matters
```

**Right:**
```
beakon context AuthService.login
↓ receive: anchor code + callers + callees
↓ now you know exactly which files matter
cat auth/service.go     ← only if you need the full file
grep "session" auth/    ← only if you need text search
```

Three tools. Layered, not competing:

| Tool | Use when |
|------|----------|
| `grep` | You know the text exists and where |
| `cat` | You already know the file |
| `beakon` | You don't know where to look |

---

## Supported Languages

- Go
- TypeScript / TSX
- JavaScript / JSX
- Python

All parsing uses [Tree-sitter](https://tree-sitter.github.io/tree-sitter/) via [`go-tree-sitter`](https://github.com/smacker/go-tree-sitter). No language servers. No daemons.

---

## How It Works

```
Repository
    ↓
internal/repo       — scan files, detect languages
    ↓
internal/symbols    — Tree-sitter parsing, symbol + call extraction
    ↓
internal/indexer    — orchestrate: full index, incremental update, watch mode
    ↓
internal/index      — read/write .beakon/ JSON files
internal/graph      — build + query call graph
    ↓
cmd/beakon          — CLI (Cobra)
```

The index lives in `.beakon/` as JSON. Queries read from the index — never scan source files. Both directions of the call graph are precomputed at index time so all lookups are O(1).

### Incremental Updates

When a file changes, Beakon surgically replaces that file's contribution to the index. No full re-scan. Target: **<200ms** per update.

### Watch Mode

```bash
beakon watch
```

Monitors your repo with fsnotify. Two-level debounce (50ms flush / 500ms max) keeps the index current as you edit without thrashing on rapid saves.

---

## Storage Layout

```
.beakon/
├── meta.json           index metadata
├── symbols.json        flat list of all symbols
├── map.json            directory → symbol names
├── files/
│   └── *.json          one record per source file
└── graph/
    ├── calls_from.json  symbol → what it calls
    └── calls_to.json    symbol → what calls it
```

Add `.beakon/` to your `.gitignore`.

---

## Performance

| Operation | Target |
|-----------|--------|
| Any query | <100ms |
| Incremental file update | <200ms |
| Full index (medium repo) | <30s |

---

## Using with Claude Code

Beakon is designed to work alongside Claude Code. Add this to your `CLAUDE.md`:

```markdown
## Navigation Model

Three tools. Each has a distinct purpose.

    grep        → find text you know exists
    cat         → read a file you already located
    beakon      → navigate when you don't know where to look

## First Thing Every Session

    go build -o beakon ./cmd/beakon
    ./beakon index
    ./beakon map --human

## Before Modifying Any Symbol

    ./beakon context <symbol> --human

This shows you the full blast radius before you touch anything.
```

Claude will use `beakon context` before exploring files, cutting irrelevant token usage and improving accuracy.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

```bash
git clone https://github.com/beakon/beakon
cd beakon
go mod tidy
go build -o beakon ./cmd/beakon
go test ./...
```

Key docs:
- `SPEC.md` — data structures and storage layout
- `ARCHITECTURE.md` — pipeline and package responsibilities
- `TASKS.md` — what is done, what is next
- `REPO_RULES.md` — invariants that must not be broken

---

## License

MIT
