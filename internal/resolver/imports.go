package resolver

import (
	"bufio"
	"bytes"
	"strings"
)

// resolvedImport holds the package info for one import alias or direct binding.
type resolvedImport struct {
	Package    string
	Stdlib     string // "yes" | "no" | "unknown"
	Resolution string // "resolved" | "unresolved"
	Reason     string // why unresolved
}

// importMap maps qualifier/symbol name → resolvedImport.
// Both qualifier-based ("json" → encoding/json) and direct-binding ("Path" → pathlib) entries live here.
type importMap map[string]resolvedImport

// parseImports extracts all import bindings from source for the given language.
func parseImports(language string, src []byte) importMap {
	switch language {
	case "go":
		return parseGoImports(src)
	case "typescript", "javascript":
		return parseTSImports(src)
	case "python":
		return parsePyImports(src)
	case "rust":
		return parseRustImports(src)
	case "java", "groovy":
		return parseJavaImports(src)
	case "ruby":
		return parseRubyImports(src)
	}
	return importMap{}
}

// --- Go ---

func parseGoImports(src []byte) importMap {
	m := importMap{}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "import (" {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}

		var raw string
		if inBlock {
			raw = line
		} else if strings.HasPrefix(line, "import ") {
			raw = strings.TrimPrefix(line, "import ")
			raw = strings.TrimSpace(raw)
		} else {
			continue
		}

		// Strip trailing comment
		if i := strings.Index(raw, "//"); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
		if raw == "" {
			continue
		}

		// Dot import — unresolvable
		if strings.HasPrefix(raw, ". ") {
			pkg := unquote(strings.TrimPrefix(raw, ". "))
			qualifier := lastPathSegment(pkg)
			m[qualifier] = resolvedImport{
				Package:    pkg,
				Stdlib:     goStdlib(pkg),
				Resolution: "unresolved",
				Reason:     "dot_import",
			}
			continue
		}

		// Aliased import: alias "pkg/path"
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) == 2 {
			alias := parts[0]
			pkg := unquote(parts[1])
			m[alias] = resolvedImport{Package: pkg, Stdlib: goStdlib(pkg), Resolution: "resolved"}
			continue
		}

		// Plain import: "pkg/path"
		pkg := unquote(raw)
		if pkg != "" {
			qualifier := lastPathSegment(pkg)
			m[qualifier] = resolvedImport{Package: pkg, Stdlib: goStdlib(pkg), Resolution: "resolved"}
		}
	}
	return m
}

func goStdlib(pkg string) string {
	// No dot in first segment = stdlib
	first := pkg
	if i := strings.Index(pkg, "/"); i >= 0 {
		first = pkg[:i]
	}
	if strings.Contains(first, ".") {
		return "no"
	}
	return "yes"
}

// --- TypeScript / JavaScript ---

func parseTSImports(src []byte) importMap {
	m := importMap{}
	lines := joinTSImportBlocks(src)
	for _, line := range lines {
		// import ... from 'pkg'
		if strings.HasPrefix(line, "import ") && strings.Contains(line, " from ") {
			parseTSImportFrom(line, m)
			continue
		}
		// const X = require('pkg') or const { X } = require('pkg')
		if strings.Contains(line, "require(") {
			parseTSRequire(line, m)
			continue
		}
	}
	return m
}

// joinTSImportBlocks pre-processes TypeScript/JavaScript source to collapse
// multi-line import statements onto a single line. For example:
//
//	import {
//	  readFile,
//	  writeFile,
//	} from 'node:fs/promises'
//
// becomes: import { readFile, writeFile, } from 'node:fs/promises'
func joinTSImportBlocks(src []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(src))
	var result []string
	var pending strings.Builder
	depth := 0 // brace depth inside an accumulating import

	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if pending.Len() > 0 {
			// Inside a multi-line import — accumulate
			pending.WriteByte(' ')
			pending.WriteString(trimmed)
			depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			// Done when brace depth returns to 0 and we've seen 'from'
			if depth <= 0 && strings.Contains(pending.String(), " from ") {
				result = append(result, strings.TrimSpace(pending.String()))
				pending.Reset()
				depth = 0
			}
			continue
		}

		// Single-line import — already complete
		if strings.HasPrefix(trimmed, "import ") && strings.Contains(trimmed, " from ") {
			result = append(result, trimmed)
			continue
		}

		// Start of a multi-line import: `import {` or `import type {` with no `from` yet
		if strings.HasPrefix(trimmed, "import ") && strings.Contains(trimmed, "{") && !strings.Contains(trimmed, " from ") {
			pending.WriteString(trimmed)
			depth = strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			continue
		}

		// require() lines — pass through unchanged
		if strings.Contains(trimmed, "require(") {
			result = append(result, trimmed)
			continue
		}
	}
	return result
}

func parseTSImportFrom(line string, m importMap) {
	// Extract package from `from 'pkg'` or `from "pkg"`
	fromIdx := strings.LastIndex(line, " from ")
	if fromIdx < 0 {
		return
	}
	pkgRaw := strings.TrimSpace(line[fromIdx+6:])
	pkgRaw = strings.TrimSuffix(pkgRaw, ";")
	pkg := unquote(pkgRaw)
	if pkg == "" {
		return
	}
	stdlib := tsStdlib(pkg)
	spec := strings.TrimSpace(line[len("import "):fromIdx])

	// import * as alias from 'pkg'
	if strings.HasPrefix(spec, "* as ") {
		alias := strings.TrimPrefix(spec, "* as ")
		m[alias] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		return
	}

	// import defaultExport from 'pkg' (no braces)
	if !strings.Contains(spec, "{") {
		spec = strings.TrimSpace(spec)
		if spec != "" && !strings.HasPrefix(spec, "type") {
			// Could be "DefaultExport" or "DefaultExport, { X }"
			parts := strings.SplitN(spec, ",", 2)
			def := strings.TrimSpace(parts[0])
			if def != "" {
				m[def] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
			}
			if len(parts) > 1 {
				parseTSNamedBindings(parts[1], pkg, stdlib, m)
			}
		}
		return
	}

	// import { X, Y as Z } from 'pkg'
	parseTSNamedBindings(spec, pkg, stdlib, m)
}

func parseTSNamedBindings(spec, pkg, stdlib string, m importMap) {
	start := strings.Index(spec, "{")
	end := strings.LastIndex(spec, "}")
	if start < 0 || end < 0 || end <= start {
		return
	}
	bindings := spec[start+1 : end]
	for _, b := range strings.Split(bindings, ",") {
		b = strings.TrimSpace(b)
		b = strings.TrimPrefix(b, "type ")
		if b == "" {
			continue
		}
		// Handle "X as Y" → use Y as the local name
		if asParts := strings.Split(b, " as "); len(asParts) == 2 {
			local := strings.TrimSpace(asParts[1])
			m[local] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		} else {
			m[b] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		}
	}
}

func parseTSRequire(line string, m importMap) {
	// Extract pkg from require('pkg') or require("pkg")
	start := strings.Index(line, "require(")
	if start < 0 {
		return
	}
	rest := line[start+8:]
	end := strings.Index(rest, ")")
	if end < 0 {
		return
	}
	pkg := unquote(strings.TrimSpace(rest[:end]))
	if pkg == "" {
		return
	}
	stdlib := tsStdlib(pkg)

	// const { X, Y } = require('pkg')
	if strings.Contains(line[:start], "{") {
		lhs := line[:start]
		bStart := strings.Index(lhs, "{")
		bEnd := strings.LastIndex(lhs, "}")
		if bStart >= 0 && bEnd > bStart {
			parseTSNamedBindings(lhs[bStart:bEnd+1], pkg, stdlib, m)
			return
		}
	}

	// const X = require('pkg')
	if eqIdx := strings.Index(line[:start], "="); eqIdx >= 0 {
		lhs := strings.TrimSpace(line[:eqIdx])
		// Strip const/let/var
		for _, kw := range []string{"const ", "let ", "var "} {
			lhs = strings.TrimPrefix(lhs, kw)
		}
		lhs = strings.TrimSpace(lhs)
		if lhs != "" && !strings.Contains(lhs, "{") {
			m[lhs] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		}
	}
}

func tsStdlib(pkg string) string {
	if strings.HasPrefix(pkg, "node:") {
		return "yes"
	}
	// Conventional Node builtins — could be shadowed by npm package
	nodeBuiltins := map[string]bool{
		"fs": true, "path": true, "os": true, "url": true, "http": true,
		"https": true, "crypto": true, "stream": true, "events": true,
		"util": true, "buffer": true, "child_process": true, "cluster": true,
		"dns": true, "net": true, "readline": true, "repl": true,
		"tls": true, "vm": true, "worker_threads": true, "zlib": true,
		"assert": true, "console": true, "process": true, "timers": true,
		"perf_hooks": true, "module": true, "querystring": true, "string_decoder": true,
	}
	if nodeBuiltins[pkg] {
		return "unknown"
	}
	return "no"
}

// --- Python ---

func parsePyImports(src []byte) importMap {
	m := importMap{}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// from X import *
		if strings.HasPrefix(line, "from ") && strings.Contains(line, " import *") {
			pkg := strings.TrimSpace(strings.Split(strings.TrimPrefix(line, "from "), " import")[0])
			// Wildcard — unresolved; add a sentinel so callee lookup returns unresolved
			m["__wildcard__"+pkg] = resolvedImport{
				Package:    pkg,
				Stdlib:     pyStdlib(pkg),
				Resolution: "unresolved",
				Reason:     "wildcard_import",
			}
			continue
		}

		// from X import Y, Z or from X import (Y, Z)
		if strings.HasPrefix(line, "from ") && strings.Contains(line, " import ") {
			parsePyFromImport(line, m)
			continue
		}

		// import X or import X as Y or import X, Y
		if strings.HasPrefix(line, "import ") {
			parsePyDirectImport(line, m)
			continue
		}
	}
	return m
}

func parsePyFromImport(line string, m importMap) {
	// "from X import Y, Z as W"
	parts := strings.SplitN(strings.TrimPrefix(line, "from "), " import ", 2)
	if len(parts) != 2 {
		return
	}
	pkg := strings.TrimSpace(parts[0])
	stdlib := pyStdlib(pkg)
	bindings := strings.Trim(strings.TrimSpace(parts[1]), "()")
	for _, b := range strings.Split(bindings, ",") {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if asParts := strings.Split(b, " as "); len(asParts) == 2 {
			local := strings.TrimSpace(asParts[1])
			m[local] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		} else {
			m[b] = resolvedImport{Package: pkg, Stdlib: stdlib, Resolution: "resolved"}
		}
	}
}

func parsePyDirectImport(line string, m importMap) {
	// "import X, Y as Z"
	rest := strings.TrimPrefix(line, "import ")
	for _, part := range strings.Split(rest, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if asParts := strings.Split(part, " as "); len(asParts) == 2 {
			pkg := strings.TrimSpace(asParts[0])
			alias := strings.TrimSpace(asParts[1])
			m[alias] = resolvedImport{Package: pkg, Stdlib: pyStdlib(pkg), Resolution: "resolved"}
		} else {
			// "import os.path" → qualifier is "os"
			qualifier := strings.SplitN(part, ".", 2)[0]
			m[qualifier] = resolvedImport{Package: part, Stdlib: pyStdlib(part), Resolution: "resolved"}
		}
	}
}

var pyStdlibModules = map[string]bool{
	"os": true, "sys": true, "re": true, "json": true, "math": true,
	"time": true, "datetime": true, "pathlib": true, "collections": true,
	"itertools": true, "functools": true, "typing": true, "io": true,
	"abc": true, "threading": true, "multiprocessing": true, "subprocess": true,
	"urllib": true, "http": true, "email": true, "html": true, "xml": true,
	"csv": true, "logging": true, "unittest": true, "argparse": true,
	"dataclasses": true, "enum": true, "contextlib": true, "copy": true,
	"shutil": true, "tempfile": true, "hashlib": true, "base64": true,
	"struct": true, "socket": true, "asyncio": true, "inspect": true,
	"ast": true, "warnings": true, "traceback": true, "string": true,
	"random": true, "textwrap": true, "pprint": true, "array": true,
	"bisect": true, "heapq": true, "queue": true, "weakref": true,
	"gc": true, "platform": true, "signal": true, "stat": true,
	"glob": true, "fnmatch": true, "linecache": true, "tokenize": true,
	"types": true, "operator": true, "pickle": true, "shelve": true,
	"sqlite3": true, "zlib": true, "gzip": true, "bz2": true, "lzma": true,
	"zipfile": true, "tarfile": true, "configparser": true, "secrets": true,
	"getpass": true, "getopt": true, "optparse": true, "fileinput": true,
	"atexit": true, "builtins": true, "cProfile": true, "profile": true,
	"timeit": true, "trace": true, "tracemalloc": true, "dis": true,
	"code": true, "codeop": true, "keyword": true, "token": true,
	"numbers": true, "decimal": true, "fractions": true, "statistics": true,
}

func pyStdlib(pkg string) string {
	// Use the top-level module name
	top := strings.SplitN(pkg, ".", 2)[0]
	if pyStdlibModules[top] {
		return "yes"
	}
	return "no"
}

// --- Rust ---

func parseRustImports(src []byte) importMap {
	m := importMap{}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "use ") {
			parseRustUse(strings.TrimPrefix(line, "use "), m)
		}
	}
	return m
}

func parseRustUse(stmt string, m importMap) {
	stmt = strings.TrimSuffix(stmt, ";")
	stmt = strings.TrimSpace(stmt)

	// use crate::foo as bar;
	if asParts := strings.SplitN(stmt, " as ", 2); len(asParts) == 2 {
		alias := strings.TrimSpace(asParts[1])
		pkg := strings.TrimSpace(asParts[0])
		m[alias] = resolvedImport{Package: pkg, Stdlib: rustStdlib(pkg), Resolution: "resolved"}
		return
	}

	// use std::collections::{HashMap, BTreeMap};
	if braceStart := strings.Index(stmt, "::{"); braceStart >= 0 {
		prefix := stmt[:braceStart]
		rest := stmt[braceStart+3:]
		end := strings.Index(rest, "}")
		if end < 0 {
			return
		}
		bindings := rest[:end]
		stdlib := rustStdlib(prefix)
		for _, b := range strings.Split(bindings, ",") {
			b = strings.TrimSpace(b)
			if b == "" || b == "self" {
				continue
			}
			if asParts := strings.SplitN(b, " as ", 2); len(asParts) == 2 {
				local := strings.TrimSpace(asParts[1])
				m[local] = resolvedImport{Package: prefix, Stdlib: stdlib, Resolution: "resolved"}
			} else {
				m[b] = resolvedImport{Package: prefix, Stdlib: stdlib, Resolution: "resolved"}
			}
		}
		return
	}

	// use std::collections::HashMap;  (simple path)
	lastSep := strings.LastIndex(stmt, "::")
	if lastSep >= 0 {
		pkg := stmt[:lastSep]
		symbol := stmt[lastSep+2:]
		m[symbol] = resolvedImport{Package: pkg, Stdlib: rustStdlib(pkg), Resolution: "resolved"}
		return
	}

	// use serde;  (bare crate)
	m[stmt] = resolvedImport{Package: stmt, Stdlib: rustStdlib(stmt), Resolution: "resolved"}
}

func rustStdlib(pkg string) string {
	top := strings.SplitN(pkg, "::", 2)[0]
	switch top {
	case "std", "core", "alloc":
		return "yes"
	}
	return "no"
}

// --- Java / Groovy ---

func parseJavaImports(src []byte) importMap {
	m := importMap{}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "import ") {
			continue
		}
		stmt := strings.TrimPrefix(line, "import ")
		stmt = strings.TrimSuffix(stmt, ";")
		stmt = strings.TrimSpace(stmt)
		stmt = strings.TrimPrefix(stmt, "static ")

		// Wildcard import
		if strings.HasSuffix(stmt, ".*") {
			pkg := strings.TrimSuffix(stmt, ".*")
			m["__wildcard__"+pkg] = resolvedImport{
				Package:    pkg,
				Stdlib:     javaStdlib(pkg),
				Resolution: "unresolved",
				Reason:     "wildcard_import",
			}
			continue
		}

		// Exact import: java.util.HashMap → symbol: HashMap, package: java.util
		lastDot := strings.LastIndex(stmt, ".")
		if lastDot >= 0 {
			symbol := stmt[lastDot+1:]
			pkg := stmt[:lastDot]
			m[symbol] = resolvedImport{Package: pkg, Stdlib: javaStdlib(pkg), Resolution: "resolved"}
		}
	}
	return m
}

func javaStdlib(pkg string) string {
	if strings.HasPrefix(pkg, "java.") || strings.HasPrefix(pkg, "javax.") || strings.HasPrefix(pkg, "sun.") {
		return "yes"
	}
	return "no"
}

// --- Ruby ---

func parseRubyImports(src []byte) importMap {
	m := importMap{}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// require 'pkg' or require "pkg"
		if strings.HasPrefix(line, "require ") || strings.HasPrefix(line, "require('") {
			raw := strings.TrimPrefix(line, "require ")
			raw = strings.TrimPrefix(raw, "(")
			raw = strings.TrimSuffix(raw, ")")
			pkg := unquote(raw)
			if pkg == "" {
				continue
			}
			// Map package to its conventional module constant (e.g., "json" → "JSON")
			qualifier := rubyQualifier(pkg)
			m[qualifier] = resolvedImport{Package: pkg, Stdlib: rubyStdlib(pkg), Resolution: "resolved"}
			// Also add the raw package name as qualifier
			if qualifier != pkg {
				m[pkg] = resolvedImport{Package: pkg, Stdlib: rubyStdlib(pkg), Resolution: "resolved"}
			}
		}
	}
	return m
}

var rubyStdlibModules = map[string]bool{
	"json": true, "net/http": true, "uri": true, "open-uri": true, "csv": true,
	"yaml": true, "date": true, "time": true, "pathname": true, "fileutils": true,
	"tempfile": true, "digest": true, "base64": true, "openssl": true, "zlib": true,
	"set": true, "stringio": true, "ostruct": true, "struct": true, "monitor": true,
	"thread": true, "mutex": true, "forwardable": true, "singleton": true,
	"comparable": true, "enumerable": true, "logger": true, "pp": true,
	"benchmark": true, "coverage": true, "objspace": true, "rbconfig": true,
	"securerandom": true, "shellwords": true, "socket": true, "io": true,
	"cgi": true, "erb": true, "rexml/document": true,
}

func rubyStdlib(pkg string) string {
	if rubyStdlibModules[pkg] {
		return "yes"
	}
	return "unknown"
}

// rubyQualifier maps a require path to its conventional constant name.
func rubyQualifier(pkg string) string {
	known := map[string]string{
		"json":         "JSON",
		"date":         "Date",
		"time":         "Time",
		"pathname":     "Pathname",
		"fileutils":    "FileUtils",
		"set":          "Set",
		"yaml":         "YAML",
		"base64":       "Base64",
		"digest":       "Digest",
		"openssl":      "OpenSSL",
		"uri":          "URI",
		"net/http":     "Net::HTTP",
		"stringio":     "StringIO",
		"ostruct":      "OpenStruct",
		"securerandom": "SecureRandom",
		"cgi":          "CGI",
	}
	if q, ok := known[pkg]; ok {
		return q
	}
	// Convert foo_bar → FooBar
	parts := strings.Split(pkg, "/")
	last := parts[len(parts)-1]
	return snakeToPascal(last)
}

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]))
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

// --- Helpers ---

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func lastPathSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
