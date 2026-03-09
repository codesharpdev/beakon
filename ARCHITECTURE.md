# Beakon — Architecture

## Pipeline

```
Repository
    ↓
internal/repo       — scan files, detect languages
    ↓
internal/symbols    — Tree-sitter parsing, symbol + call extraction
    ↓
internal/resolver   — enrich external calls: imports → lockfiles → stdlib
    ↓
internal/indexer    — orchestrate: full index, incremental update, watch mode
    ↓
internal/index      — read/write .beakon/ JSON files
internal/graph      — build + query call graph
    ↓
internal/context    — assemble complete LLM context bundles
    ↓
cmd/beakon          — CLI commands (Cobra)
```

---

## Package Responsibilities

### pkg/
Shared data structures only. No logic.

    types.go    BeakonNode, CallEdge, FileIndex, TraceStep, ExternalCallee, ExternalIndex, NodeID()

No other packages in pkg/. If unsure, put it in internal/.

---

### internal/repo/
Repository discovery. No parsing, no indexing.

    scan.go     Walk repo, skip ignored dirs, return []SourceFile

Respects .gitignore patterns (simple prefix/suffix matching).
Knows about: file extensions, skip directories.
Does NOT know about: symbols, graph, index.

---

### internal/symbols/
Tree-sitter parsing and symbol extraction.

    parse.go    Tree-sitter language wiring (18 languages)
    extract.go  Walk AST → []BeakonNode + []CallEdge

One function per language family: extractGo, extractTS, extractPython, extractRust, extractJava, etc.
Knows about: AST nodes, pkg.BeakonNode, pkg.CallEdge.
Does NOT know about: files, storage, graph, resolver.

---

### internal/resolver/
External dependency enrichment. Runs after symbol extraction, before indexing.

    builtins.go     IsBuiltin(language, symbol) — language-specific builtin sets
    imports.go      ParseImports(language, src) → importMap (qualifier → package info)
    lockfile.go     ReadLockfile(root, filePath, language) → version + devOnly info
    enrich.go       Enrich(root, filePath, language, src, calls) → []CallEdge (enriched)

Pipeline:
1. Parse all import statements from source
2. For each call edge:
   - Drop if builtin (len, print, console.log, etc.)
   - Resolve qualifier via importMap → get Package, Stdlib
   - Look up version from lockfile (go.mod, package.json, poetry.lock, etc.)
   - Annotate: Package, Stdlib, Version, DevOnly, Resolution, Reason

Knows about: pkg.CallEdge, stdlib lists, lockfile formats.
Does NOT know about: BeakonNode, graph, indexer, CLI.

---

### internal/graph/
Call graph construction and traversal.

    build.go    Build(edges) → CallsFrom + CallsTo
                BuildExternal(edges) → ExternalIndex
                Write / Read{From,To,External}
                Trace(symbol, from) → []string          (forward BFS, names only)
                TraceRich(symbol, from, symIndex, root)  (forward BFS, with snippets)
                Impact(symbol, to) → []string            (reverse BFS, names only)
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

All writes to symbols.json, map.json, calls_from.json, calls_to.json are atomic
(write to .tmp file, then os.Rename).

Knows about: pkg.FileIndex, pkg.BeakonNode, disk paths.
Does NOT know about: graph, parser, CLI.

---

### internal/indexer/
Orchestration layer. The only package that combines repo + symbols + resolver + graph + index.

    index_repo.go   Run(root) — full parallel reindex (NumCPU goroutines)
    update.go       UpdateFile(root, path) — surgical single-file update (Strategy B)
    watch.go        Watcher — fsnotify + two-level debounce (50ms flush / 500ms max)

Knows about: all internal packages.
Does NOT know about: CLI, cobra.

---

### internal/context/
Highest-level internal package. Assembles the complete context bundle an LLM needs
to reason about a symbol.

    engine.go   Engine.Assemble(query) → Bundle

Pipeline:
1. Lazy-load all index data on first call (symIdx, from, to, extIdx)
2. Find anchor symbol (exact match, then suffix match)
3. Fetch anchor source code (live read)
4. Look up direct callees in calls_from
5. For each callee: fetch source (internal) or enrichment metadata (external)
6. Look up direct callers in calls_to
7. Fetch each caller source code
8. Collect unique files
9. Estimate tokens (chars / 4)

Knows about: internal/graph, internal/index, internal/code, pkg
Does NOT know about: indexer, repo, symbols, resolver, CLI

---

### internal/code/
Source code extraction for the show command and context assembly.

    fetch.go    Fetch(file, start, end) → Block{File, Start, End, Code}

Reads live from source files. Never reads from index.

---

### cmd/beakon/
CLI entry point. No business logic.

    main.go     Cobra commands: index, watch, map, trace, explain,
                callers, deps, show, search, context, impact

Each command:
1. Gets repoRoot from os.Getwd()
2. Calls one internal package function
3. Formats output (JSON default, --human flag)

---

## Dependency Rules

Allowed imports (each layer may only import layers below it):

```
cmd/beakon
    → internal/context
    → internal/indexer
    → internal/graph
    → internal/index
    → internal/code
    → pkg

internal/context
    → internal/graph
    → internal/index
    → internal/code
    → pkg

internal/indexer
    → internal/repo
    → internal/symbols
    → internal/resolver
    → internal/graph
    → internal/index
    → pkg

internal/resolver → pkg (+ stdlib only for file I/O)
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

.beakon/nodes/ mirrors the repository structure.
One JSON file per source file (path slashes + dots → underscores).
Example: auth/service.go → auth_service_go.json

.beakon/graph/ holds precomputed bidirectional edges.
Both directions are built at index time so callers queries are O(1).
external.json stores enrichment metadata for external callees.

---

## Incremental Indexing (Strategy B)

When a file changes:

1. Compute new SHA1 hash
2. Compare with stored hash in nodes/*.json
3. If same → skip
4. If different:
   a. Parse new version → new symbols + calls
   b. Enrich external calls via resolver
   c. Write new nodes/<file>.json
   d. Load ALL other nodes/*.json
   e. Merge: other files' symbols + new file's symbols
   f. Rewrite symbols.json, map.json, calls_from.json, calls_to.json, external.json

Step (d-f) is O(files in repo), not O(repo LOC).
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

## External Dependency Enrichment

Runs at index time inside internal/resolver.
Enriched data stored in .beakon/graph/external.json.
Context engine reads external.json to include enrichment in context bundles.

Enrichment per call edge:
- Package  — resolved import path
- Stdlib   — "yes" | "no" | "unknown"
- Version  — pinned version from lockfile
- DevOnly  — bool pointer (nil = unknown); true for devDependencies
- Resolution — "resolved" | "unresolved"
- Reason   — why unresolved: "dot_import", "wildcard_import", "no_import_found"
- Hint     — extra information for agent consumption
