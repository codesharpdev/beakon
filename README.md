# Beakon

**Shared code intelligence infrastructure for AI agents.**

Beakon parses repositories with Tree-sitter, builds a bidirectional call graph, enriches every external dependency with version and provenance metadata, and assembles complete context bundles — so AI agents reason about the right code, not random grep results.

> **Long-term vision:** Instead of every AI coding tool rebuilding repository understanding from scratch, Beakon becomes shared infrastructure. Index once. Query from anywhere.

---

## The Problem

AI agents navigating codebases face a structural problem: they don't know what they don't know.

The typical approach:
```
grep "login" → open 20 files → guess which one matters → hit token limits
```

This wastes context on irrelevant code, misses the files that actually matter, and gives the agent no signal about external dependencies or blast radius.

## The Solution

```bash
beakon context AuthService.Login
```

Returns:
- The symbol's full source code
- Source of every function it calls — including whether each is stdlib, a pinned third-party dependency, or dev-only
- Source of every function that calls it
- All files involved
- Token estimate

One command. Complete picture.

---

## Installation

Requirements: **Go 1.21+**

```bash
git clone https://github.com/codesharpdev/beakon
cd beakon
go build -o beakon ./cmd/beakon
```

---

## Quick Start

```bash
# Index your repository
beakon index

# Get complete context for a symbol
beakon context AuthService.Login --human

# See what would break if you change a symbol
beakon impact createJWT --human

# See the repo structure
beakon map --human
```

---

## Commands

| Goal | Command |
|------|---------|
| Index the repository | `beakon index` |
| Watch for changes (incremental) | `beakon watch` |
| **Complete LLM context bundle** | `beakon context <symbol>` |
| Blast radius of a change | `beakon impact <symbol>` |
| Repo structure overview | `beakon map` |
| Search for a symbol | `beakon search <text>` |
| Show symbol source | `beakon show <symbol>` |
| Trace execution flow | `beakon trace <symbol>` |
| Explain a feature end-to-end | `beakon explain <symbol>` |
| Who calls this function | `beakon callers <symbol>` |
| What this function depends on | `beakon deps <symbol>` |

All commands output **JSON by default** (machine-readable for agents) and readable text with `--human`.

---

## External Dependency Enrichment

A differentiating feature: Beakon doesn't just map internal calls. Every external call edge is enriched at index time:

```json
{
  "from": "AuthService.Login",
  "to": "bcrypt.GenerateFromPassword",
  "package": "golang.org/x/crypto/bcrypt",
  "stdlib": "no",
  "version": "v0.17.0",
  "dev_only": false,
  "resolution": "resolved"
}
```

Enrichment includes:
- **Package** — the import it came from
- **Stdlib** — yes / no / unknown
- **Version** — pinned version from lockfile (`go.mod`, `package.json`, `poetry.lock`, `Cargo.toml`, `Gemfile.lock`)
- **DevOnly** — whether it's a dev/test-only dependency
- **Resolution** — resolved or unresolved (with reason: `dot_import`, `wildcard_import`, etc.)

This tells agents not just *what* a function calls, but *what it depends on* — enabling security audits, upgrade analysis, and dependency-aware reasoning.

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

All parsing uses [Tree-sitter](https://tree-sitter.github.io/tree-sitter/). No language servers. No daemons. No runtime dependencies.

---

## How It Works

```
Repository
    ↓
internal/repo       — scan files, detect languages
    ↓
internal/symbols    — Tree-sitter AST parsing → symbols + call edges
    ↓
internal/resolver   — enrich external calls: imports → lockfiles → stdlib detection
    ↓
internal/indexer    — orchestrate: full index, incremental update, watch mode
    ↓
internal/index      — read/write .beakon/ JSON files
internal/graph      — build + query bidirectional call graph
    ↓
internal/context    — assemble complete LLM context bundles
    ↓
cmd/beakon          — CLI (Cobra)
```

### Storage Layout

```
.beakon/
├── meta.json           index metadata (version, file count, symbol count)
├── symbols.json        flat list of all symbols
├── map.json            directory → symbol names
├── nodes/
│   └── *.json          one FileIndex per source file
└── graph/
    ├── calls_from.json  symbol → what it calls
    ├── calls_to.json    symbol → what calls it
    └── external.json    enriched external dependency metadata
```

Add `.beakon/` to your `.gitignore`.

### Incremental Updates

When a file changes, Beakon surgically replaces that file's contribution to the index. SHA1 hash comparison skips unchanged files. All writes are atomic (temp file + rename). Target: **<200ms** per update.

### Watch Mode

```bash
beakon watch
```

Monitors your repo with fsnotify. Two-level debounce (50ms flush, 500ms max) keeps the index current without thrashing during editor save + format + lint cycles.

---

## Performance

| Operation | Target |
|-----------|--------|
| Any query command | <100ms |
| Incremental file update | <200ms |
| Full index (medium repo) | <30s |

Indexing is parallelized across all CPU cores.

---

## Using with Claude Code

Beakon is designed to work alongside [Claude Code](https://claude.ai/claude-code). Add this to your `CLAUDE.md`:

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

This shows the full blast radius before you touch anything.
```

With this setup, Claude uses `beakon context` before exploring files — cutting irrelevant token usage and giving the agent accurate dependency and caller/callee context.

---

## Agent Workflow

**Without Beakon:**
```
grep "createJWT" → open 12 files → miss the callers → wrong fix
```

**With Beakon:**
```
beakon context createJWT
↓ anchor: createJWT source
↓ callers: AuthService.Login, TokenRefresher.Refresh
↓ deps: jwt.Sign (stdlib: no, version: v5.2.0)
↓ files: auth/service.go, api/token.go
→ open exactly those 2 files
→ correct fix, minimum tokens
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide.

```bash
git clone https://github.com/codesharpdev/beakon
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

[MIT](LICENSE)
