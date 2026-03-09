# Beakon

### The code intelligence layer for AI agents.

Beakon parses repositories with Tree-sitter, builds a bidirectional call graph, enriches every external dependency with version and provenance metadata, and assembles complete context bundles on demand — so AI agents reason about the right code, not random grep results.

> **Long-term vision:** Instead of every AI coding tool rebuilding repository understanding from scratch, Beakon becomes shared infrastructure. Index once. Query from anywhere.

[![npm](https://img.shields.io/npm/v/@codesharpdev/beakon)](https://www.npmjs.com/package/@codesharpdev/beakon)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](#license)

---

## Install

```bash
npm install -g @codesharpdev/beakon
```

Correct binary for your OS and architecture downloaded automatically. No Go toolchain required.

---

## Demo

> Demo GIF coming soon. Try it yourself:

```bash
cd your-repo
beakon index
beakon context AuthService.Login --human
```

```
 ANCHOR  auth/service.go

  func (s *AuthService) Login(email, password string) (*Session, error) {
      user, err := s.db.FindByEmail(email)
      if err != nil { return nil, ErrNotFound }
      if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
          return nil, ErrInvalidCredentials
      }
      token, err := createJWT(user.ID, s.cfg.JWTSecret)
      return &Session{Token: token, UserID: user.ID}, nil
  }

 CALLS  3 symbols

  db.FindByEmail                internal   auth/db.go:42
  bcrypt.CompareHashAndPassword external   golang.org/x/crypto  v0.17.0  stdlib:no
  createJWT                     internal   auth/token.go:18

 CALLED BY  2 symbols

  api.PostLogin                 api/auth_handler.go:55
  TestLoginSuccess              auth/service_test.go:12

 FILES  auth/service.go · auth/db.go · auth/token.go

 ~1,840 tokens
```

One command. Complete picture. Zero guessing.

---

## Why Beakon

Every AI coding tool today does the same thing: grep for a keyword, open 10–20 files, and hope the relevant code is in there. It isn't a workflow problem — it's a structural one. **Agents have no map.**

**Without Beakon:**
```
grep "createJWT"
→ open 14 files
→ miss the callers in api/
→ wrong fix, retry
```

**With Beakon:**
```
beakon context createJWT
→ anchor source + 2 callers + jwt dep at v5.2.0
→ open exactly those 2 files
→ correct fix, minimum tokens
```

The difference isn't speed. It's that the agent understands the shape of the code before it writes a single line.

### Why not RAG / embeddings?

Vector search finds *semantically similar* code. Beakon gives you *structurally exact* relationships — what this function calls, what calls it, what external packages it depends on at what pinned version. These are different questions. Embeddings can't tell you blast radius or resolve a call to `bcrypt.GenerateFromPassword → golang.org/x/crypto v0.17.0 · stdlib:no · dev:false`. Beakon can.

### What makes it different

|  | grep / ripgrep | RAG / embeddings | LSP | Beakon |
|--|----------------|-----------------|-----|--------|
| Finds text | ✓ | — | — | ✓ |
| Semantic similarity | — | ✓ | — | — |
| Bidirectional call graph | — | — | partial | ✓ |
| External dep + version metadata | — | — | — | ✓ |
| 18 languages, no language server | — | — | per-language daemon | ✓ |
| Agent-ready JSON output | — | — | — | ✓ |
| Blast radius analysis | — | — | — | ✓ |

---

## Quick Example

```bash
# What would break if I change createJWT?
beakon impact createJWT --human

# Trace execution from an HTTP handler down
beakon trace api.PostLogin --human

# Repo structure at a glance
beakon map --human

# Search by name
beakon search "session" --human
```

| Goal | Command |
|------|---------|
| Index the repository | `beakon index` |
| Watch for changes | `beakon watch` |
| Complete LLM context bundle | `beakon context <symbol>` |
| Blast radius of a change | `beakon impact <symbol>` |
| Trace execution flow | `beakon trace <symbol>` |
| Repo structure overview | `beakon map` |
| Search for a symbol | `beakon search <text>` |
| Who calls this function | `beakon callers <symbol>` |
| What this function depends on | `beakon deps <symbol>` |

All commands output **JSON by default** (pipe directly to agents) and readable text with `--human`.

---

## Dep Enrichment

A unique capability: every external call edge is resolved back to its lockfile at index time.

```json
{
  "from": "AuthService.Login",
  "to": "bcrypt.GenerateFromPassword",
  "package": "golang.org/x/crypto/bcrypt",
  "stdlib": false,
  "version": "v0.17.0",
  "dev_only": false,
  "resolution": "resolved"
}
```

Supports `go.mod` · `package.json` · `poetry.lock` · `Cargo.toml` · `Gemfile.lock`

This lets agents reason about security, upgrades, and dependency-aware changes — not just navigation.

---

## Adopting at Scale

### The infrastructure picture

```
beakon watch              ← runs in background, <200ms per file change
       │
       ├── Claude Code     beakon context <symbol>
       ├── Cursor          beakon impact <symbol>
       ├── your CI         beakon map
       └── any agent       beakon search <text>

All querying the same index. Built once. Always current.
```

### Agent instructions

Drop this into your `CLAUDE.md`, `.cursorrules`, or system prompt. The key is teaching the agent *when* to reach for beakon vs grep vs cat — they are layered, not competing.

````markdown
## Code Navigation

Three tools. Each has a distinct purpose.

    grep        → find text you know exists
    cat         → read a file you already located
    beakon      → navigate when you don't know where to look

WRONG:
    grep "login" → open 20 files → guess which one matters

CORRECT:
    beakon context AuthService.Login --human
    ↓ anchor source + callers + callees (with dep metadata)
    ↓ now you know exactly which files matter
    cat auth/service.go        ← only if you need the full file
    grep "session" auth/       ← only if you need text search

Before modifying any symbol:
    beakon context <symbol> --human   # understand it
    beakon impact <symbol> --human    # see what breaks

Never open files speculatively. beakon context tells you which
files matter before you read a single one.
````

### Watch mode

```bash
beakon watch   # stays running, incremental updates <200ms per save
```

### CI

```bash
beakon index
beakon impact <changed-symbol> --human
```

---

## Languages

18 languages. Zero language servers. Zero daemons. All parsing via [Tree-sitter](https://tree-sitter.github.io/tree-sitter/).

| Language | Extensions |
|----------|------------|
| ![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white&style=flat-square) | `.go` |
| ![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white&style=flat-square) | `.ts` `.tsx` |
| ![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?logo=javascript&logoColor=black&style=flat-square) | `.js` `.jsx` |
| ![Python](https://img.shields.io/badge/Python-3776AB?logo=python&logoColor=white&style=flat-square) | `.py` |
| ![Rust](https://img.shields.io/badge/Rust-000000?logo=rust&logoColor=white&style=flat-square) | `.rs` |
| ![Java](https://img.shields.io/badge/Java-ED8B00?logo=openjdk&logoColor=white&style=flat-square) | `.java` |
| ![C](https://img.shields.io/badge/C-A8B9CC?logo=c&logoColor=black&style=flat-square) | `.c` `.h` |
| ![C++](https://img.shields.io/badge/C++-00599C?logo=cplusplus&logoColor=white&style=flat-square) | `.cpp` `.cc` `.cxx` `.hpp` |
| ![C#](https://img.shields.io/badge/C%23-239120?logo=csharp&logoColor=white&style=flat-square) | `.cs` |
| ![Ruby](https://img.shields.io/badge/Ruby-CC342D?logo=ruby&logoColor=white&style=flat-square) | `.rb` |
| ![Kotlin](https://img.shields.io/badge/Kotlin-7F52FF?logo=kotlin&logoColor=white&style=flat-square) | `.kt` `.kts` |
| ![Swift](https://img.shields.io/badge/Swift-F05138?logo=swift&logoColor=white&style=flat-square) | `.swift` |
| ![PHP](https://img.shields.io/badge/PHP-777BB4?logo=php&logoColor=white&style=flat-square) | `.php` |
| ![Scala](https://img.shields.io/badge/Scala-DC322F?logo=scala&logoColor=white&style=flat-square) | `.scala` |
| ![Elixir](https://img.shields.io/badge/Elixir-4B275F?logo=elixir&logoColor=white&style=flat-square) | `.ex` `.exs` |
| ![OCaml](https://img.shields.io/badge/OCaml-EC6813?logo=ocaml&logoColor=white&style=flat-square) | `.ml` `.mli` |
| ![Elm](https://img.shields.io/badge/Elm-1293D8?logo=elm&logoColor=white&style=flat-square) | `.elm` |
| ![Groovy](https://img.shields.io/badge/Groovy-4298B8?logo=apachegroovy&logoColor=white&style=flat-square) | `.groovy` |

---

## Performance

| Operation | Target |
|-----------|--------|
| Any query | <100ms |
| Incremental file update | <200ms |
| Full index (medium repo) | <30s |

Indexing is parallelized across all CPU cores.

---

## Support Beakon

Beakon is free, open infrastructure. If it saves your agents time every day:

- **Star the repo** — helps other developers find it
- **Share it** — post in your team Slack, Discord, or AI tooling community
- **Sponsor** — [github.com/sponsors/codesharpdev](https://github.com/sponsors/codesharpdev) — keeps the project active

Building a product on top of Beakon — an IDE plugin, agent framework, dev tool? Reach out. Happy to help.

---

## Contributing

```bash
git clone https://github.com/codesharpdev/beakon
cd beakon
go mod tidy
go build -o beakon ./cmd/beakon
go test ./...
```

---

## License

[MIT](LICENSE)
