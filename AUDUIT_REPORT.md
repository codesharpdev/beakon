---
  CodeIndex — Capability Audit Report

  ---
  1. IMPLEMENTED FEATURES

  Symbol Extraction

  Files: internal/symbols/extract.go (372 lines), internal/symbols/parse.go (44 lines)

  Tree-sitter-based AST parsing for 4 languages. No regex anywhere.

  ┌────────────┬───────────┬─────────┬─────────┬────────────┐
  │  Language  │ Functions │ Methods │ Classes │ Call Edges │
  ├────────────┼───────────┼─────────┼─────────┼────────────┤
  │ Go         │ ✓         │ ✓       │ ✓       │ ✓          │
  ├────────────┼───────────┼─────────┼─────────┼────────────┤
  │ TypeScript │ ✓         │ ✓       │ ✓       │ ✓          │
  ├────────────┼───────────┼─────────┼─────────┼────────────┤
  │ JavaScript │ ✓         │ ✓       │ ✓       │ ✓          │
  ├────────────┼───────────┼─────────┼─────────┼────────────┤
  │ Python     │ ✓         │ ✓       │ ✓       │ ✓          │
  └────────────┴───────────┴─────────┴─────────┴────────────┘

  Each language has its own extractor (extractGo, extractTS, extractPython) producing []CodeIndexNode + []CallEdge. Call edges are extracted directly from the AST — no heuristics.

  Call Graph

  File: internal/graph/build.go (239 lines)

  Bidirectional graph precomputed at index time. Stored as two JSON files:
  - .codeindex/graph/calls_from.json — what each symbol calls
  - .codeindex/graph/calls_to.json — who calls each symbol

  Both directions are O(1) lookup at query time.

  Trace: BFS from any symbol through calls_from, with cycle detection. TraceRich adds live source snippets (6 lines each).

  Index Storage

  File: internal/index/write.go (197 lines)

  JSON-only flat file storage under .codeindex/:
  .codeindex/
  ├── meta.json           — index metadata
  ├── symbols.json        — flat symbol list
  ├── map.json            — dir → symbols (architecture overview)
  ├── files/*.json        — one FileIndex per source file
  └── graph/
      ├── calls_from.json
      └── calls_to.json

  Features: hash-based incremental skipping (NeedsUpdate), parallel ReadAll with mutex, DeleteFile for removed files.

  Full Indexer

  File: internal/indexer/index_repo.go (156 lines)

  8-worker parallel pool. Collects errors per file (non-fatal). Rebuilds symbols.json, map.json, and graph after parallel file processing. Returns full stats (files indexed/skipped, symbol count, duration, errors).

  Incremental Indexer (Strategy B)

  File: internal/indexer/update.go (182 lines)

  Single-file update path: parse changed file → load all existing file indexes → replace changed file's data → rebuild globals. O(files), not O(LOC). Handles deletes.

  Watch Mode

  File: internal/indexer/watch.go (179 lines)

  fsnotify-based recursive watcher with two-level debounce: 50ms idle OR 500ms max staleness. Calls UpdateFile per dirty path. Auto-triggers full index if .codeindex/ doesn't exist.

  Context Engine

  File: internal/context/engine.go (218 lines)

  The core LLM API. Given a symbol name, assembles:
  - Anchor: symbol source code
  - Callees: direct dependencies with source
  - Callers: direct callers with source
  - Files: deduplicated file list
  - Token estimate: chars / 4

  Supports partial name matching ("login" → "AuthService.Login").

  All 10 CLI Commands

  File: cmd/codeindex/main.go (496 lines)

  Every command implemented with JSON (default) and --human output modes:

  ┌─────────┬───────────────────────────────────────────────────────────┐
  │ Command │                       What it does                        │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ index   │ Full parallel reindex                                     │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ watch   │ Continuous incremental updates                            │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ map     │ Directory → symbols overview                              │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ context │ Complete LLM bundle (anchor + callers + callees + source) │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ callers │ Who calls this symbol                                     │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ deps    │ What this symbol calls                                    │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ trace   │ BFS call chain with inline snippets                       │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ explain │ Trace + unique files involved                             │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ show    │ Full source of a symbol                                   │
  ├─────────┼───────────────────────────────────────────────────────────┤
  │ search  │ Substring match across symbol names                       │
  └─────────┴───────────────────────────────────────────────────────────┘

  ---
  2. PARTIALLY IMPLEMENTED

  Atomic Writes

  Status: Documented, not enforced.

  SPEC.md specifies atomic writes. The code in index/write.go and graph/build.go calls writeJSON() which writes directly to the target file. If the process crashes mid-write, corrupted JSON remains and the index is broken. The fix (write
  to .tmp then os.Rename) is a 3-line change per write site, but it's not done.

  Concurrent Write Safety

  Status: Partially safe.

  ReadAll in index/write.go uses a mutex. But update.go's rebuildGlobal() — called from the watcher — has no mutex guarding the global files. Simultaneous file change events can trigger concurrent rebuildGlobal() calls, causing write races
   on symbols.json, map.json, and the graph files.

  Signal Handling in Watch Mode

  Status: Commented out.

  watch.go has this literally in the code:
  // signal.Notify(ch, os.Interrupt)
  Ctrl+C behavior is undefined. The watcher may not shut down cleanly.

  Error Recovery in Worker Pool

  Status: Goroutines have no recover().

  A panic in a parser goroutine (e.g., malformed source file hits nil pointer) silently exits the goroutine. The WaitGroup still decrements, but no error is reported. The result may be silently incomplete.

  ---
  3. MISSING CAPABILITIES

  Test Suite

  Zero test files exist. This is the most critical gap. TASKS.md lists 6 Phase 4 items (parser unit tests, graph tests, storage tests, incremental tests, integration tests, benchmarks) — all TODO. For a tool intended to be trusted by AI
  agents, this is a production blocker.

  impact Command

  Listed in TASKS.md as Phase 6 TODO. Reverse BFS from a symbol: "if I change X, what else breaks?" The graph already supports this (calls_to.json gives you callers up the tree) — the BFS traversal just needs to go the opposite direction.
  The infrastructure is there; the command is not.

  .gitignore Integration

  The scanner skips a hardcoded list of directories (.git, node_modules, vendor, etc.) but ignores .gitignore. On any real project with custom ignore rules, the index will include unwanted files and miss nothing it shouldn't miss. TASKS.md
   Phase 6 marks this TODO.

  Config File

  No .codeindex/config.yaml. Worker count is hardcoded to 8. Language extensions are hardcoded. Skip patterns are hardcoded. For production use, these need to be configurable per project.

  Progress Output During Indexing

  The index command runs silently until completion. On a large repo this looks like a hang. TASKS.md Phase 6 marks this TODO.

  Cross-File Type Resolution

  Call edges store the callee name as a string (e.g., "validatePassword"). If the same name exists in multiple files, the graph has no way to disambiguate. There is no type-aware resolution — the current approach is "best effort name
  matching," which will produce false edges in realistic codebases.

  Semantic/Intent Layer

  No capability exists to answer "what feature does this symbol belong to?" or "what is the purpose of this module?" The explain command traces calls but does not reason about intent. This is the capability gap between CodeIndex and a true
   "LSP for AI agents."

  ---
  4. ARCHITECTURE OVERVIEW

  ┌─────────────────────────────────────┐
  │           cmd/codeindex             │  CLI (Cobra, 10 commands)
  │        Thin wrapper only            │
  └────────────────┬────────────────────┘
                   │
       ┌───────────┴───────────┐
       │   internal/indexer    │  Orchestration layer
       │  (index, update, watch)│
       └──────┬────────┬───────┘
              │        │
     ┌─────────┘        └──────────────┐
     │                                 │
  ┌──┴──────────┐   ┌─────────────┐  ┌─────────────┐
  │internal/repo│   │internal/    │  │internal/    │
  │ (scanner)   │   │symbols      │  │index        │
  │             │   │(Tree-sitter)│  │(JSON storage)│
  └─────────────┘   └──────┬──────┘  └──────┬──────┘
                           │                │
                    ┌──────┴──────┐  ┌──────┴──────┐
                    │internal/    │  │internal/    │
                    │graph        │  │context      │
                    │(call graph) │  │(LLM engine) │
                    └─────────────┘  └─────────────┘
                           │
                    ┌──────┴──────┐
                    │internal/code│
                    │(source fetch)│
                    └─────────────┘
                           │
                    ┌──────┴──────┐
                    │  pkg/types  │
                    │(shared types)│
                    └─────────────┘

  Data flow:
  repo scan → file list
           → Tree-sitter parse → CodeIndexNode[] + CallEdge[]
           → parallel write → .codeindex/files/*.json
           → graph build → calls_from.json + calls_to.json
           → symbol merge → symbols.json + map.json

  query → load from disk → assemble bundle → output JSON/human

  All queries are stateless reads from disk. No in-memory cache. No server process.

  ---
  5. DESIGN RISKS

  Risk 1: Name-Based Call Resolution (HIGH)

  Call edges use raw string names. "Login" could exist in AuthService, UserService, and AdminService. The graph connects them all as if they're the same. In large real-world codebases, this degrades the context engine's precision
  significantly. The fix requires qualified names or type-aware resolution.

  Risk 2: No Atomic Writes (HIGH)

  Index corruption is possible on any unclean shutdown. Until writeJSON uses write-to-temp + rename, the index is not crash-safe. Tools that AI agents depend on must be crash-safe.

  Risk 3: Concurrent Write Race in Watch Mode (HIGH)

  Two rapid file saves trigger two concurrent rebuildGlobal() calls with no serialization. The final state of symbols.json and graph files is a coin flip. This is a real-world scenario (auto-save, code formatters).

  Risk 4: O(files) Rebuild on Every Change (MEDIUM)

  UpdateFile reloads every file index on disk to rebuild globals. At 10,000 files this is ~10,000 JSON reads per change event. The debouncer reduces frequency, but at scale this will exceed the 200ms incremental update target.

  Risk 5: Hardcoded Worker Count (LOW)

  8 workers is reasonable for small/medium repos. On a 1-core CI machine it's too many; on a 32-core workstation it's too few. Not configurable.

  Risk 6: Live Source Reads in Queries (LOW)

  show, trace, context, and explain all read source files live from disk at query time. For local dev this is fine. If CodeIndex ever serves a remote repo, this assumption breaks entirely.

  ---
  6. NEXT 5 IMPROVEMENTS

  Ranked by impact on the core mission (AI agents querying code instead of scanning files):

  #1 — Fix Concurrent Write Safety + Atomic Writes

  Why now: Before anything else, the index must be trustworthy. An AI agent that gets corrupted context silently is worse than no agent at all.

  What: Add a write mutex to update.go's rebuildGlobal. Change writeJSON to write to a .tmp file then os.Rename. Two focused changes, ~20 lines total.

  #2 — Write the Test Suite

  Why now: Every subsequent feature added without tests risks regressing the features that already work. Phase 4 is entirely TODO. Start with:
  - symbols/extract_test.go — parse sample Go/TS/Python, assert nodes + edges
  - graph/build_test.go — build graph from known edges, assert BFS correctness
  - context/engine_test.go — assemble bundle from test fixture, assert fields

  The testdata/sample_repo fixture already exists; tests can be written against it immediately.

  #3 — Implement impact Command

  Why now: This is the highest-value missing command for the AI agent use case. "What does changing X affect?" is the most common question before a refactor. The graph already supports reverse BFS; the command just needs to be written.

  codeindex impact AuthService.Login
  # → UserController.Login depends on this
  # → AdminController.Login depends on this
  # → 2 test files reference callers

  #4 — .gitignore Integration

  Why now: Without it, CodeIndex indexes test fixtures, generated files, vendor copies, and build artifacts. These pollute the symbol index with noise that degrades context quality for AI agents. Parse and respect .gitignore files during
  scanning.

  #5 — Qualify Call Edges by Receiver/Package

  Why now: Name disambiguation is the prerequisite for the context engine becoming truly reliable. The fix doesn't require full type inference — just store the qualified call name (AuthService.validatePassword instead of validatePassword)
  during extraction by propagating the enclosing type/package context during AST walks. This is already done for symbol names; it needs to be applied to call edge targets.

  ---
  7. FEATURES TO AVOID FOR NOW

  Full LSP Server

  LSP requires a persistent process, incremental document sync, per-client state, and strict protocol compliance. CodeIndex's stateless query model is an architectural strength — it lets AI agents query without a daemon. Adding LSP would
  require a complete architectural shift before the core is solid. Wait until test coverage is >80% and the robustness phase is complete.

  Distributed / Remote Indexing

  The live source read assumption is baked into context, show, trace, and explain. Distributing the index without distributing the source is impossible without a major redesign. This is a premature optimization — the target is medium repos
   in local environments.

  Semantic Embeddings / Vector Search

  Adding vector embeddings for semantic symbol search would require a dependency on an ML runtime or an external API, making the tool stateful and network-dependent. The current substring search is fast and dependency-free. Semantic search
   should wait until the structural graph is complete and tested.

  Web / GUI Interface

  The CLI + JSON output is the correct interface for AI agents. A web UI serves human developers but adds a server process, authentication concerns, and a frontend build system. The CLAUDE.md instructions explicitly frame this as "LSP for
  AI agents" — not for human browsing.

  Plugin / Language Extension System

  Only 4 languages are supported, and all 4 are fully functional. A plugin system to add languages adds significant complexity (interface design, versioning, loading mechanism) before it's clear what the extension points should actually
  look like. Add Rust or Java as first-class built-ins when needed; build a plugin system only when there are 6+ languages.

  ---
  Summary: CodeIndex has a solid, complete core (2,235 lines, 12 files, all 10 commands working). The architecture is clean and well-layered. The critical gaps are test coverage, write safety, and the impact command. Fix those three and
  the system is genuinely production-ready for its stated purpose.