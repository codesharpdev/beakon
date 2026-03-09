# Beakon — Architecture

## Pipeline

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
cmd/beakon       — CLI commands (Cobra)
```

---

## Package Responsibilities

### pkg/
Shared data structures only. No logic.

    types.go    BeakonNode, CallEdge, FileIndex, TraceStep, NodeID()

No other packages in pkg/. If unsure, put it in internal/.

---

### internal/repo/
Repository discovery. No parsing, no indexing.

    scan.go     Walk repo, skip ignored dirs, return []SourceFile

Knows about: file extensions, skip directories.
Does NOT know about: symbols, graph, index.

---

### internal/symbols/
Tree-sitter parsing and symbol extraction.

    parse.go    Tree-sitter language wiring
    extract.go  Walk AST → []BeakonNode + []CallEdge

One function per language family: extractGo, extractTS, extractPython.
Knows about: AST nodes, pkg.BeakonNode, pkg.CallEdge.
Does NOT know about: files, storage, graph.

---

### internal/graph/
Call graph construction and traversal.

    build.go    Build(edges) → CallsFrom + CallsTo
                Write / ReadFrom / ReadTo
                Trace(symbol, from) → []string          (names only)
                TraceRich(symbol, from, symIndex, root)  (with snippets)
                ExplainResult struct

Knows about: pkg.CallEdge, pkg.TraceStep, graph JSON files.
Does NOT know about: indexer, repo scanner, CLI.

---

### internal/index/
Filesystem storage layer for .beakon/.

    write.go    Init, Write, Read, ReadAll, DeleteFile
                WriteSymbols, ReadSymbols
                WriteMap, ReadMap
                WriteMeta, ReadMeta
                NeedsUpdate(root, file, hash) bool

Knows about: pkg.FileIndex, pkg.BeakonNode, disk paths.
Does NOT know about: graph, parser, CLI.

---

### internal/indexer/
Orchestration layer. The only package that combines repo + symbols + graph + index.

    index_repo.go   Run(root) — full parallel reindex
    update.go       UpdateFile(root, path) — surgical single-file update (Strategy B)
    watch.go        Watcher — fsnotify + 50ms debounce loop

Knows about: all internal packages.
Does NOT know about: CLI, cobra.

---

### internal/code/
Source code extraction for the show command.

    fetch.go    Fetch(file, start, end) → Block{File, Start, End, Code}

Reads live from source files. Never reads from index.

---

### cmd/beakon/
CLI entry point. No business logic.

    main.go     Cobra commands: index, watch, map, trace, explain,
                callers, deps, show, search

Each command:
1. Gets repoRoot from os.Getwd()
2. Calls one internal package function
3. Formats output (JSON default, --human flag)

---

## Dependency Rules

Allowed imports (each layer may only import layers below it):

```
cmd/beakon
    → internal/indexer
    → internal/graph
    → internal/index
    → internal/code
    → pkg

internal/indexer
    → internal/repo
    → internal/symbols
    → internal/graph
    → internal/index
    → pkg

internal/graph   → pkg
internal/index   → pkg
internal/symbols → pkg
internal/repo    → (stdlib only)
internal/code    → (stdlib only)
pkg              → (stdlib only)
```

Cross-layer imports are forbidden.
Example: internal/graph must never import internal/indexer.

---

## Storage Strategy

All index data lives in .beakon/ as JSON.

Queries read from JSON files — never scan source files.
Exception: show command and snippet extraction read source files live.

.beakon/files/ mirrors the repository structure.
One JSON file per source file.

.beakon/graph/ holds precomputed bidirectional edges.
Both directions are built at index time so callers queries are O(1).

---

## Incremental Indexing (Strategy B)

When a file changes:

1. Compute new SHA1 hash
2. Compare with stored hash in files/*.json
3. If same → skip
4. If different:
   a. Parse new version → new symbols + calls
   b. Write new files/<file>.json
   c. Load ALL other files/*.json
   d. Merge: other files' symbols + new file's symbols
   e. Rewrite symbols.json, map.json, calls_from.json, calls_to.json

Step (c-e) is O(files in repo), not O(repo LOC).
Target: <200ms for typical repos.

---

## Watch Mode

Watcher (internal/indexer/watch.go):

1. fsnotify watches all repo subdirectories recursively
2. File events enter a pending map with timestamp
3. Ticker fires every 50ms — flush files older than 50ms
4. Max debounce: 500ms — flush even if events keep arriving
5. Call UpdateFile() for each flushed path
6. Emit WatchEvent on Events channel

Two-level debounce prevents:
- Burst re-indexing when editor saves + formats + lints
- Starvation of a heavily-edited file (500ms cap)

---

## CLI Output Contract

Default output: JSON to stdout.
Human output: --human flag, readable text.

JSON schema is stable — agents depend on it.
Human output is for display — format can evolve freely.

Error output always goes to stderr.
Exit code 1 on any error.

---

## internal/context/ (Context Engine)

The highest-level internal package. Assembles the complete context
bundle an LLM needs to reason about a symbol.

    engine.go   Engine.Assemble(query) → Bundle

Pipeline:
1. Find anchor symbol in symIdx
2. Fetch anchor source code (live read)
3. Look up direct callees in calls_from
4. Fetch each callee source code
5. Look up direct callers in calls_to
6. Fetch each caller source code
7. Collect unique files
8. Estimate tokens

Knows about: internal/graph, internal/index, pkg
Does NOT know about: indexer, repo, symbols, CLI

This is the package that replaces:
    grep + open files + guess architecture
