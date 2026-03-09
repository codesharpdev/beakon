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
Skips: .git, .beakon, node_modules, vendor, dist, build, __pycache__
Languages: .go .ts .tsx .js .jsx .py

### 1.2 Tree-sitter Integration
Status: DONE
Files: internal/symbols/parse.go, internal/symbols/extract.go
Languages: Go, TypeScript, JavaScript, Python
Extracts: functions, methods, classes, call edges

### 1.3 File Index Storage
Status: DONE
File: internal/index/write.go
Stores: .beakon/files/*.json, symbols.json, map.json, meta.json

### 1.4 Call Graph (Bidirectional)
Status: DONE
File: internal/graph/build.go
Stores: .beakon/graph/calls_from.json + calls_to.json
Precomputed at index time — both directions

### 1.5 Full Indexer
Status: DONE
File: internal/indexer/index_repo.go
Parallel workers (8), skips unchanged files by hash

---

## Phase 2 — CLI Commands (DONE)

### 2.1 index
Status: DONE
Runs full index, prints file + symbol count + duration

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

## Phase 4 — Tests (TODO)

### 4.1 Unit Tests — Parser
Status: TODO
File: internal/symbols/extract_test.go

Test cases:
- Go function extraction
- Go method extraction with receiver
- Go type/struct extraction
- TypeScript class + method extraction
- Python function + class extraction
- Call edge detection (Go)
- Call edge detection (TypeScript)
- Empty file → zero nodes
- File with syntax errors → no panic, zero nodes

### 4.2 Unit Tests — Graph
Status: TODO
File: internal/graph/build_test.go

Test cases:
- Build() produces correct calls_from
- Build() produces correct calls_to
- Trace() BFS order
- Trace() cycle detection (no infinite loop)
- TraceRich() enriches with file + line
- Empty graph → empty result

### 4.3 Unit Tests — Index Storage
Status: TODO
File: internal/index/write_test.go

Test cases:
- Write + Read FileIndex round-trip
- NeedsUpdate() returns true on hash change
- NeedsUpdate() returns false on same hash
- DeleteFile() removes file cleanly
- ReadAll() loads all files

### 4.4 Unit Tests — Incremental Update
Status: TODO
File: internal/indexer/update_test.go

Test cases:
- UpdateFile() on changed file → new symbols appear in index
- UpdateFile() on unchanged file → skipped
- UpdateFile() on deleted file → symbols removed from index
- UpdateFile() on new file → symbols added
- symbols.json reflects correct state after update
- calls_from.json reflects correct state after update

### 4.5 Integration Tests
Status: TODO
File: internal/indexer/integration_test.go

Test cases:
- Full index of testdata/sample_repo
- trace AuthService.Login returns correct chain
- callers createJWT returns AuthService.Login
- show AuthService.Login returns correct source block
- search "login" returns relevant symbols
- map output groups by directory correctly

### 4.6 Performance Benchmarks
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

## Phase 5 — Robustness (TODO)

### 5.1 Error Handling
Status: TODO

Requirements:
- Single file parse failure must not abort full index
- Log warning per failed file, continue
- Return partial results with Errors []string in Result
- Never panic on malformed source files

### 5.2 Concurrent Watch Safety
Status: TODO

Requirements:
- Multiple rapid saves to same file must produce one update
- Concurrent updates to different files must not corrupt index
- symbols.json write must be atomic (write to temp, rename)

Implementation:
- Use os.WriteFile with temp file + os.Rename for atomic writes
- Add mutex around global index rebuild in update.go

### 5.3 Large Repo Support
Status: TODO

Requirements:
- 50k files, 10M LOC must not OOM
- Parallel indexer must respect memory limits
- Add configurable worker count (default 8, env: CODEINDEX_WORKERS)

---

## Phase 6 — Quality of Life (TODO)

### 6.1 Progress Output
Status: TODO

During index: show live progress
Example:
    indexing... 142/380 files

Implementation:
- Optional progress reporter interface in indexer.Run()
- CLI wires in a simple stderr printer when --human is set

### 6.2 impact Command
Status: TODO
File: cmd/beakon/main.go + internal/graph/build.go

Show everything that would break if a symbol changes.
Algorithm: reverse BFS from symbol through calls_to graph.

Example:
    ./beakon impact createJWT --human

Output:
    impact: createJWT
    affected (2):
      AuthService.Login  auth/service.go:30
      UserController.Login  api/controller.go:14

### 6.3 Config File
Status: TODO
File: .beakon/config.yaml

Options:
    version: 1
    languages: [go, typescript, python]
    ignore: [vendor/, generated/]
    workers: 8

### 6.4 .gitignore Integration
Status: TODO

Read .gitignore and skip matching paths during scan.

---

## Current Priority Order

1. Phase 4 — Tests (highest priority)
2. Phase 5.2 — Atomic writes (correctness)
3. Phase 6.2 — impact command
4. Phase 5.1 — Error handling
5. Phase 6.1 — Progress output
6. Phase 6.3 — Config file

---

## How To Pick a Task

Read this file.
Find first TODO in priority order.
Read SPEC.md and ARCHITECTURE.md before implementing.
Write tests alongside implementation.
Run go test ./... before committing.

---

## Phase 7 — Context Engine (DONE)

### 7.1 Context Engine
Status: DONE
File: internal/context/engine.go

Assembles complete LLM context bundle for a symbol:
- Anchor symbol + full source code
- Direct callees + their source code
- Direct callers + their source code
- Unique files involved
- Token estimate (chars / 4)

### 7.2 context command
Status: DONE
Location: cmd/beakon/main.go

Usage:
    beakon context <symbol>
    beakon context <symbol> --human

JSON output: full Bundle struct
Human output: anchor + CALLS + CALLED BY sections with source
