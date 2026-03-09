# Language Builtins & External Package Resolution Reference
## For Beakon `kind: "external"` Callee Enrichment

---

## Design Principles

1. **Drop builtins entirely** from callee lists — they add noise without value
2. **Enrich true externals** with package metadata
3. **Use three-state stdlib** field: `"yes"` / `"no"` / `"unknown"`
4. **Surface resolution failures honestly** with `resolution` + `reason` fields
5. **Drop empty external fields** — never return `file: ""`, `start: 0`, `code: ""`

### Enriched External Shape
```json
{
  "symbol": "json.Marshal",
  "kind": "external",
  "package": "encoding/json",
  "stdlib": "yes",
  "dev_only": false,
  "version": null,
  "resolution": "resolved",
  "reason": null
}
```

### Unresolved External Shape
```json
{
  "symbol": "Button",
  "kind": "external",
  "package": "@myapp/components",
  "stdlib": "no",
  "dev_only": false,
  "version": "workspace:*",
  "resolution": "unresolved",
  "reason": "barrel_reexport",
  "hint": "see src/components/index.ts"
}
```

---

## Go

### Builtins to Drop
```
make append len cap delete copy close panic recover print println
new real imag complex clear min max
```

### Stdlib Detection
No module path = stdlib. Mechanical check, no lookup needed.
```
"encoding/json"     → stdlib: "yes"
"fmt"               → stdlib: "yes"  
"net/http"          → stdlib: "yes"
"github.com/..."    → stdlib: "no"
"golang.org/x/..."  → stdlib: "no"  (extended packages, NOT stdlib)
```

### Qualifier Resolution
Qualifier prefix maps directly to import alias:
```go
import "encoding/json"           // alias: "json"
import j "encoding/json"         // alias: "j"  
import . "encoding/json"         // dot import → resolution: "unresolved", reason: "dot_import"
```

### Hard Cases
| Pattern | Handling |
|---------|----------|
| `import . "pkg"` | `resolution: "unresolved"`, `reason: "dot_import"` |
| CGo calls | `resolution: "unresolved"`, `reason: "cgo"` |
| `//go:linkname` | `resolution: "unresolved"`, `reason: "linkname"` |

### Version Source
`go.mod` → `require` block for exact module version.

---

## Python

### Builtins to Drop
```
print len range type str int float bool list dict set tuple
input open abs all any bin chr dir divmod enumerate filter
format frozenset getattr globals hasattr hash help hex id
isinstance issubclass iter locals map max min next object
oct ord pow repr reversed round setattr slice sorted staticmethod
sum super vars zip __import__ breakpoint callable classmethod
compile complex delattr eval exec memoryview property
```

### Stdlib Detection
Ship a hardcoded set per Python version. Python 3.11+ exposes `sys.stdlib_module_names`.

Key stdlib modules (non-exhaustive):
```
os sys re json math time datetime pathlib collections itertools
functools typing io abc threading multiprocessing subprocess
urllib http email html xml csv logging unittest argparse
dataclasses enum contextlib copy shutil tempfile hashlib
base64 struct socket asyncio inspect ast warnings traceback
```

### Qualifier Resolution
```python
import json                      # alias: "json"     → package: "json"
import numpy as np               # alias: "np"        → package: "numpy"
from pathlib import Path         # direct binding     → package: "pathlib"
from os.path import *            # wildcard → resolution: "unresolved", reason: "wildcard_import"
```

### Hard Cases
| Pattern | Handling |
|---------|----------|
| `from pkg import *` | `resolution: "unresolved"`, `reason: "wildcard_import"` |
| `__import__(var)` | `resolution: "unresolved"`, `reason: "dynamic_import"` |
| `importlib.import_module(var)` | `resolution: "unresolved"`, `reason: "dynamic_import"` |

### Version Source
`requirements.txt`, `poetry.lock`, `Pipfile.lock`, `pyproject.toml`.

---

## TypeScript / JavaScript / TSX / JSX

### Builtins to Drop
```
console JSON Math Object Array String Number Boolean Promise
parseInt parseFloat isNaN isFinite decodeURI encodeURI
setTimeout setInterval clearTimeout clearInterval
fetch Symbol Map Set WeakMap WeakSet Proxy Reflect
Error TypeError ReferenceError SyntaxError
undefined null Infinity NaN eval globalThis
```

### Stdlib Detection (Node.js)
Only trust `node:` prefix as definitive:
```
"node:fs"           → stdlib: "yes"  (definitive)
"node:path"         → stdlib: "yes"  (definitive)
"fs"                → stdlib: "unknown"  (conventionally Node stdlib, but npm package could shadow)
"path"              → stdlib: "unknown"
"react"             → stdlib: "no"
```

### Import Pattern Resolution
```ts
// Named import — resolved
import { readFile } from 'node:fs/promises'
// symbol: "readFile", package: "node:fs/promises", stdlib: "yes"

// Default import — resolved
import axios from 'axios'
// symbol: "axios", package: "axios", stdlib: "no"

// Namespace import — resolved
import * as path from 'path'
// symbol: "path.*", package: "path", stdlib: "unknown"

// CJS require — resolved
const express = require('express')
const { Router } = require('express')
// symbol: "express"/"Router", package: "express", stdlib: "no"

// Dynamic import — unresolved
const lib = await import(someVar)
// resolution: "unresolved", reason: "dynamic_import"

// Barrel re-export — partially surfaced
import { Button } from '@myapp/ui'
// resolution: "unresolved", reason: "barrel_reexport", hint: "see src/ui/index.ts"
```

### dev_only Flag
Cross-reference `package.json`:
- `dependencies` → `dev_only: false`
- `devDependencies` → `dev_only: true`
- Not found → `dev_only: null`

### Hard Cases
| Pattern | Handling |
|---------|----------|
| `import(variable)` | `resolution: "unresolved"`, `reason: "dynamic_import"` |
| Barrel re-exports | `resolution: "unresolved"`, `reason: "barrel_reexport"`, `hint: nearest index.ts` |
| CJS/ESM mixed | Handle both syntaxes in Tree-sitter walker |
| Bare `fs`/`path` without `node:` | `stdlib: "unknown"` |

### Version Source
`package.json` for range, `package-lock.json` or `yarn.lock` for pinned version.

---

## Rust

### Builtins to Drop
```
println! print! eprintln! eprint! panic! assert! assert_eq! assert_ne!
vec! format! write! writeln! todo! unimplemented! unreachable!
dbg! include! include_str! include_bytes! concat! env! option_env!
Box Vec String Option Result Ok Err Some None
Drop Clone Copy Send Sync Sized Default
```

### Stdlib Detection
Crate prefix is definitive:
```
"std::collections::HashMap"   → stdlib: "yes"
"std::io::Write"              → stdlib: "yes"
"core::fmt::Display"          → stdlib: "yes"
"alloc::string::String"       → stdlib: "yes"
"serde::Serialize"            → stdlib: "no"
"tokio::runtime::Runtime"     → stdlib: "no"
```

### Qualifier Resolution
```rust
use std::collections::HashMap;    // direct binding
use serde::{Serialize, Deserialize};  // multi-binding
use tokio as async_rt;            // alias: "async_rt" → package: "tokio"
```

### Hard Cases
| Pattern | Handling |
|---------|----------|
| Macro-generated calls | `resolution: "unresolved"`, `reason: "macro_generated"` |
| `extern crate` (old style) | Handle alongside `use` statements |
| Proc macros | `resolution: "unresolved"`, `reason: "proc_macro"` |

### Version Source
`Cargo.toml` `[dependencies]` block, `Cargo.lock` for pinned version.

---

## Java

### Builtins to Drop
```
System.out.println System.err.println Object.toString Object.equals
Object.hashCode Object.getClass String.valueOf String.format
Integer.parseInt Double.parseDouble Boolean.parseBoolean
Math.abs Math.max Math.min Math.round Math.floor Math.ceil
Arrays.asList Collections.unmodifiableList
```

### Stdlib Detection
Package prefix is definitive:
```
"java.*"            → stdlib: "yes"
"javax.*"           → stdlib: "yes"
"sun.*"             → stdlib: "yes"  (internal, flag as stdlib but note it)
"com.google.*"      → stdlib: "no"
"org.springframework.*" → stdlib: "no"
```

### Qualifier Resolution
```java
import java.util.HashMap;           // direct → package: "java.util"
import java.util.*;                 // wildcard → resolution: "unresolved", reason: "wildcard_import"
import static java.lang.Math.*;     // static wildcard → same
```

### Hard Cases
| Pattern | Handling |
|---------|----------|
| Wildcard imports | `resolution: "unresolved"`, `reason: "wildcard_import"` |
| Reflection (`Class.forName`) | `resolution: "unresolved"`, `reason: "reflection"` |
| `sun.*` internal APIs | `stdlib: "yes"`, add `note: "internal_jdk_api"` |

### Version Source
`pom.xml` (Maven), `build.gradle` / `build.gradle.kts` (Gradle).

---

## Ruby

### Builtins to Drop
```
puts print p pp gets raise require_relative attr_accessor attr_reader
attr_writer include extend prepend private protected public
lambda proc method send respond_to? nil? is_a? kind_of? class
freeze frozen? dup clone object_id equal? instance_of?
```

### Stdlib Detection
Hardcoded list (partial — version dependent):
```
json net/http uri open-uri csv yaml date time pathname
fileutils tempfile digest base64 openssl zlib set
stringio ostruct struct monitor thread mutex forwardable
singleton comparable enumerable
```
Everything else → `stdlib: "unknown"` (could be stdlib or gem without type checker).

### Qualifier Resolution
Ruby is the hardest — `require` adds to global namespace:
```ruby
require 'json'        # JSON module enters global scope — no qualifier
JSON.parse(str)       # resolvable: qualifier "JSON" → package "json"

require 'date'        # Date class enters global scope
Date.today            # resolvable: qualifier "Date" → package "date"

require 'active_support/all'  # unknown what enters scope
something.present?    # resolution: "unresolved", reason: "open_namespace"
```

### Hard Cases
| Pattern | Handling |
|---------|----------|
| Bare method calls with no qualifier | `resolution: "unresolved"`, `reason: "open_namespace"` |
| `require` with variable | `resolution: "unresolved"`, `reason: "dynamic_require"` |
| Monkey patching | Fundamentally unresolvable statically |
| Autoloading (Rails) | `resolution: "unresolved"`, `reason: "autoload"` |

Ruby should document explicitly: **partial resolution only**. Qualifier-prefixed calls (module/class prefix) are resolvable. Bare method calls are not.

### Version Source
`Gemfile.lock` for pinned versions.

---

## Cross-Language Resolution Coverage Matrix

| Language | Qualifier Resolution | Stdlib Detection | Hard Cases |
|----------|---------------------|------------------|------------|
| Go | Reliable | Reliable | Dot imports, CGo |
| Rust | Reliable | Reliable | Proc macros |
| Java | Usually reliable | Reliable | Wildcard imports, reflection |
| TypeScript | Reliable | Partial (`node:` only) | Barrel re-exports, dynamic imports |
| JavaScript | Usually reliable | Partial (`node:` only) | Dynamic require, CJS/ESM mix |
| Python | Usually reliable | Reliable (hardcoded list) | Wildcard imports, dynamic import |
| Ruby | Partial | Partial | Open namespace, autoload |

---

## Implementation Prompt

Use this prompt to implement the builtin filter + external enrichment in Beakon:

---

```
You are working on Beakon, a Go CLI tool for structural code intelligence.

TASK: Implement external callee enrichment and builtin filtering across all supported languages.

CONTEXT:
Currently, `kind: "external"` callees are returned with empty fields:
  { "symbol": "make", "kind": "external", "file": "", "start": 0, "end": 0, "code": "" }

TARGET OUTPUT for true externals:
  {
    "symbol": "json.Marshal",
    "kind": "external",
    "package": "encoding/json",
    "stdlib": "yes",        // "yes" | "no" | "unknown"
    "dev_only": false,      // null if not determinable
    "version": "v1.21.0",   // null for stdlib, semver string for third-party
    "resolution": "resolved", // "resolved" | "unresolved"
    "reason": null          // why unresolved if applicable
  }

Builtins (language primitives that are NOT imports) must be dropped entirely from callee lists.

IMPLEMENTATION STEPS:

1. Create `internal/resolver/builtins.go`
   Define a map[string]map[string]bool where outer key is language name and inner
   key is builtin symbol name. Populate with builtins for:
   - go: make, append, len, cap, delete, copy, close, panic, recover, print, println, new, real, imag, complex, clear, min, max
   - python: print, len, range, type, str, int, float, bool, list, dict, set, tuple, input, open, abs, all, any, bin, chr, dir, divmod, enumerate, filter, format, frozenset, getattr, globals, hasattr, hash, help, hex, id, isinstance, issubclass, iter, locals, map, max, min, next, object, oct, ord, pow, repr, reversed, round, setattr, slice, sorted, staticmethod, sum, super, vars, zip
   - typescript/javascript: console, JSON, Math, Object, Array, String, Number, Boolean, Promise, parseInt, parseFloat, isNaN, isFinite, setTimeout, setInterval, clearTimeout, clearInterval, fetch, Symbol, Map, Set, WeakMap, WeakSet, Error, TypeError, undefined, null, globalThis, eval
   - rust: println!, print!, eprintln!, eprint!, panic!, assert!, assert_eq!, vec!, format!, Box, Vec, String, Option, Result, Ok, Err, Some, None, Drop, Clone, Copy
   - java: System, Object, String, Integer, Double, Boolean, Math, Arrays, Collections (when called as pure builtins with no import)
   - ruby: puts, print, p, pp, gets, raise, require_relative, attr_accessor, attr_reader, attr_writer, include, extend, nil?, is_a?, freeze

2. Create `internal/resolver/imports.go`
   Define interface:
   ```go
   type ImportResolver interface {
       ExtractImports(root *sitter.Node, src []byte) []Import
       ResolveQualifier(callNode *sitter.Node, imports []Import) *ResolvedPackage
   }

   type Import struct {
       Alias   string
       Package string
   }

   type ResolvedPackage struct {
       Package    string
       Stdlib     string // "yes" | "no" | "unknown"
       Resolution string // "resolved" | "unresolved"
       Reason     string // populated when unresolved
       Hint       string // optional navigation hint
   }
   ```

3. Create one file per language implementing ImportResolver:
   - `internal/resolver/go.go`     — qualifier = import alias, stdlib = no dot in path
   - `internal/resolver/python.go` — handle `import X`, `from X import Y`, wildcard → unresolved
   - `internal/resolver/ts.go`     — handle named/default/namespace/CJS require, node: prefix for stdlib
   - `internal/resolver/rust.go`   — use/extern crate, std::/core::/alloc:: = stdlib
   - `internal/resolver/java.go`   — java.*/javax.* = stdlib, wildcard → unresolved
   - `internal/resolver/ruby.go`   — qualifier-prefixed only, bare calls → unresolved

4. Create `internal/resolver/lockfile.go`
   Define interface:
   ```go
   type LockfileReader interface {
       Version(packageName string) (string, bool)
   }
   ```
   Implement for: go.mod, package.json+lockfile, requirements.txt/poetry.lock, Cargo.toml+Cargo.lock, pom.xml, Gemfile.lock
   Look for lockfiles by walking up from the indexed file's directory to repo root.

5. Update `CallEdge` in `pkg/types.go` to add optional enrichment fields:
   ```go
   type CallEdge struct {
       From string
       To   string
       // Enrichment fields — populated for kind: "external" only
       Package    string `json:"package,omitempty"`
       Stdlib     string `json:"stdlib,omitempty"`     // "yes"|"no"|"unknown"
       DevOnly    *bool  `json:"dev_only,omitempty"`
       Version    string `json:"version,omitempty"`
       Resolution string `json:"resolution,omitempty"` // "resolved"|"unresolved"
       Reason     string `json:"reason,omitempty"`
       Hint       string `json:"hint,omitempty"`
   }
   ```

6. Update each language's parser/extractor to:
   a. Filter out builtin symbols using the builtins map before appending to Calls
   b. For remaining external calls, run ImportResolver to enrich the CallEdge
   c. Run LockfileReader to populate Version field for non-stdlib packages

7. Update query response serialization to include enrichment fields in callee output.
   Drop `file`, `start`, `end`, `code` fields for `kind: "external"` — they are always empty and waste tokens.

CONSTRAINTS:
- Enrichment happens at INDEX TIME, not query time — pay the cost once
- Never fail indexing due to resolution failure — always fall back to resolution: "unresolved"
- External builtins (make, append, etc.) are dropped silently with no log entry
- The existing CallEdge JSON storage format must remain backward compatible —
  treat missing enrichment fields as resolution: "unresolved" on read

TEST CASES to verify:
- Go: `json.Marshal` → package: "encoding/json", stdlib: "yes"
- Go: `zap.New` → package: "go.uber.org/zap", stdlib: "no", version from go.mod
- Go: `make` → dropped from callee list entirely
- Python: `requests.get` → package: "requests", stdlib: "no"
- Python: `len` → dropped
- Python: wildcard import call → resolution: "unresolved", reason: "wildcard_import"
- TS: `readFile` from `node:fs/promises` → stdlib: "yes"
- TS: `useState` from `react` → stdlib: "no", dev_only: false
- TS: `describe` from `vitest` → stdlib: "no", dev_only: true
- TS: dynamic import → resolution: "unresolved", reason: "dynamic_import"
- Rust: `HashMap` from `std::collections` → stdlib: "yes"
- Rust: `Serialize` from `serde` → stdlib: "no"
```

---

## Quick Reference: Resolution Reason Codes

| Reason | Languages | Meaning |
|--------|-----------|---------|
| `wildcard_import` | Python, Java | `from x import *` or `import x.*` |
| `dynamic_import` | Python, TS, JS | Import path is a runtime value |
| `dot_import` | Go | `import . "pkg"` pollutes namespace |
| `barrel_reexport` | TS, JS | Symbol re-exported through index file |
| `open_namespace` | Ruby | Bare method call, no qualifier |
| `dynamic_require` | Ruby, JS | `require(variable)` |
| `reflection` | Java | `Class.forName()` or similar |
| `macro_generated` | Rust | Call site generated by macro expansion |
| `proc_macro` | Rust | Procedural macro attribution |
| `autoload` | Ruby | Rails/Zeitwerk autoloading |
| `cgo` | Go | C interop via CGo |
| `linkname` | Go | `//go:linkname` directive |
