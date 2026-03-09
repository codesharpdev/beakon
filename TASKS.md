# Beakon — Tasks

## Status Legend

DONE     implemented and working
TODO     not yet implemented
PARTIAL  started but incomplete

---

## Phase 1 — Core Index (DONE)

### 1.1 Repository Scanner
Status: DONE
File: internal/repo/scan.go
Skips: .git, .beakon, node_modules, vendor, dist, build, __pycache__, .venv, venv
Languages: .go .ts .tsx .js .jsx .py .rs .java .c .h .cpp .cc .cxx .hpp .cs .rb .kt .kts .swift .php .scala .ex .exs .ml .mli .elm .groovy
Respects .gitignore patterns

### 1.2 Tree-sitter Integration
Status: DONE
Files: internal/symbols/parse.go, internal/symbols/extract.go
Languages: Go, TypeScript, JavaScript, Python, Rust, Java, C, C++, C#, Ruby, Kotlin, Swift, PHP, Scala, Elixir, OCaml, Elm, Groovy
Extracts: functions, methods, classes, call edges

### 1.3 File Index Storage
Status: DONE
File: internal/index/write.go
Stores: .beakon/nodes/*.json, symbols.json, map.json, meta.json
All writes atomic (temp file + os.Rename)

### 1.4 Call Graph (Bidirectional)
Status: DONE
File: internal/graph/build.go
Stores: .beakon/graph/calls_from.json + calls_to.json
Precomputed at index time — both directions

### 1.5 Full Indexer
Status: DONE
File: internal/indexer/index_repo.go
Parallel workers (runtime.NumCPU()), skips unchanged files by hash

---

## Phase 2 — CLI Commands (DONE)

### 2.1 index
Status: DONE
Runs full index, prints file + symbol count + duration
Shows unsupported extension summary

### 2.2 map
Status: DONE
Reads map.json, groups by directory

### 2.3 trace
Status: DONE
JSON: chain of symbol names
--human: rich output with file:line + 6-line code snippet

### 2.4 explain
Status: DONE
Full flow summary + files involved
JSON + --human modes

### 2.5 callers
Status: DONE
Reads calls_to.json — O(1) lookup

### 2.6 deps
Status: DONE
Reads calls_from.json — O(1) lookup

### 2.7 show
Status: DONE
Reads symbol location from index, fetches live source

### 2.8 search
Status: DONE
Case-insensitive substring match against symbols.json

### 2.9 context
Status: DONE
File: cmd/beakon/main.go + internal/context/engine.go
Complete LLM context bundle: anchor + callers + callees (with external enrichment) + files + token estimate

### 2.10 impact
Status: DONE
File: cmd/beakon/main.go + internal/graph/build.go
Reverse BFS through calls_to graph — shows everything that depends on a symbol transitively

---

## Phase 3 — Incremental + Watch (DONE)

### 3.1 Incremental Update (Strategy B)
Status: DONE
File: internal/indexer/update.go
Surgical: replace one file's contribution, rewrite global indexes
Handles: file change, file deletion

### 3.2 Watch Mode
Status: DONE
File: internal/indexer/watch.go
fsnotify, 50ms debounce, 500ms max debounce
Auto-runs initial index if .beakon missing

---

## Phase 4 — External Dependency Enrichment (DONE)

### 4.1 Builtin Filtering
Status: DONE
File: internal/resolver/builtins.go
Drops language-native builtins from call edges
Supports: Go, Python, TypeScript, JavaScript, Rust, Java, Groovy, Ruby

### 4.2 Import Parsing
Status: DONE
File: internal/resolver/imports.go
Parses all import forms for each language
Go: dot imports, aliases, grouped imports
TypeScript/JavaScript: named imports, default imports, require()
Python: from/import, wildcard, aliases
Rust: use statements, path aliases
Java/Groovy: wildcard imports, fully qualified imports
Ruby: require with qualifier mapping

### 4.3 Lockfile Version Pinning
Status: DONE
File: internal/resolver/lockfile.go
Go: go.mod
Node: package.json + package-lock.json
Python: requirements.txt + poetry.lock
Rust: Cargo.toml
Ruby: Gemfile.lock
Detects devDependencies for TypeScript/JavaScript

### 4.4 Call Edge Enrichment
Status: DONE
File: internal/resolver/enrich.go
Annotates external calls with Package, Stdlib, Version, DevOnly, Resolution, Reason
Stored in .beakon/graph/external.json

---

## Phase 5 — Context Engine (DONE)

### 5.1 Context Engine
Status: DONE
File: internal/context/engine.go
Assembles complete LLM context bundle for a symbol:
- Anchor symbol + full source code
- Direct callees: internal (source) or external (enrichment metadata)
- Direct callers + their source code
- Unique files involved
- Token estimate (chars / 4)

---

## Phase 6 — Tests (PARTIAL)

### 6.1 Unit Tests — Parser
Status: PARTIAL
File: internal/symbols/extract_test.go
Done: Go function/method/class extraction, TypeScript, Python, Rust
TODO: All 18 languages covered, edge cases (empty file, syntax errors)

### 6.2 Unit Tests — Graph
Status: PARTIAL
File: internal/graph/build_test.go
Done: Build(), Trace(), bidirectional lookup
TODO: Cycle detection, TraceRich, Impact

### 6.3 Unit Tests — Index Storage
Status: PARTIAL
File: internal/index/write_test.go
Done: Write + Read round-trip, NeedsUpdate
TODO: DeleteFile, ReadAll, atomic write verification

### 6.4 Unit Tests — Incremental Update
Status: PARTIAL
File: internal/indexer/update_test.go
Done: UpdateFile on changed/unchanged file
TODO: Deletion, new file, external.json state after update

### 6.5 Unit Tests — Resolver
Status: PARTIAL
File: internal/resolver/resolver_test.go
Done: Import parsing for Go, TypeScript, Python, Rust, Java, Ruby
Done: Builtin filtering, version pinning, dev-only detection
TODO: Edge cases for all 18 languages

### 6.6 Unit Tests — Context Engine
Status: PARTIAL
File: internal/context/engine_test.go
Done: Anchor lookup, callee/caller assembly, token estimate, external enrichment
TODO: Missing symbol error, partial match, file deduplication

### 6.7 Integration Tests
Status: PARTIAL
File: internal/indexer/integration_test.go
Done: Full index + update + context assembly end-to-end
TODO: Watch mode, impact command, enrichment round-trip

### 6.8 Performance Benchmarks
Status: TODO
File: internal/indexer/bench_test.go

Benchmarks:
- BenchmarkFullIndex — medium repo (~500 files)
- BenchmarkIncrementalUpdate — single file change
- BenchmarkTraceQuery — trace on 50-node chain
- BenchmarkSearchQuery — search across 1000 symbols

Pass criteria:
- Full index < 30s
- Incremental < 200ms
- Queries < 100ms

---

## Phase 7 — Robustness (TODO)

### 7.1 Error Handling
Status: TODO

Requirements:
- Single file parse failure must not abort full index
- Log warning per failed file, continue
- Return partial results with Errors []string in Result
- Never panic on malformed source files

### 7.2 Concurrent Watch Safety
Status: TODO

Requirements:
- Multiple rapid saves to same file must produce one update
- Concurrent updates to different files must not corrupt index
- symbols.json write must be atomic (already done — verify under concurrency)

Implementation:
- Add mutex around global index rebuild in update.go

### 7.3 Large Repo Support
Status: TODO

Requirements:
- 50k files, 10M LOC must not OOM
- Parallel indexer must respect memory limits
- Add configurable worker count (default NumCPU, env: BEAKON_WORKERS)

---

## Phase 8 — Quality of Life (TODO)

### 8.1 Progress Output
Status: TODO

During index: show live progress
Example:
    indexing... 142/380 files

Implementation:
- Optional progress reporter interface in indexer.Run()
- CLI wires in a simple stderr printer when --human is set

### 8.2 Config File
Status: TODO
File: .beakon/config.yaml

Options:
    version: 1
    languages: [go, typescript, python]
    ignore: [vendor/, generated/]
    workers: 8

### 8.3 .gitignore Integration (Full)
Status: TODO

Current: simple prefix/suffix pattern matching
Target: full .gitignore spec compliance (glob patterns, negation, directory rules)

---

## Current Priority Order

1. Phase 6.8 — Benchmarks (validate performance targets)
2. Phase 7.2 — Concurrent watch safety (correctness)
3. Phase 7.1 — Error handling
4. Phase 6 remaining — Test coverage
5. Phase 8.1 — Progress output
6. Phase 8.2 — Config file

---

## How To Pick a Task

Read this file.
Find first TODO in priority order.
Read SPEC.md and ARCHITECTURE.md before implementing.
Write tests alongside implementation.
Run go test ./... before committing.
