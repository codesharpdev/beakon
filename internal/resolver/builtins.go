package resolver

import "strings"

// builtinSets maps language → set of builtin symbol names that should be dropped.
var builtinSets = map[string]map[string]bool{
	"go": setOf(
		"make", "append", "len", "cap", "delete", "copy", "close",
		"panic", "recover", "print", "println", "new", "real", "imag",
		"complex", "clear", "min", "max",
	),
	"python": setOf(
		"print", "len", "range", "type", "str", "int", "float", "bool",
		"list", "dict", "set", "tuple", "input", "open", "abs", "all",
		"any", "bin", "chr", "dir", "divmod", "enumerate", "filter",
		"format", "frozenset", "getattr", "globals", "hasattr", "hash",
		"help", "hex", "id", "isinstance", "issubclass", "iter", "locals",
		"map", "max", "min", "next", "object", "oct", "ord", "pow",
		"repr", "reversed", "round", "setattr", "slice", "sorted",
		"staticmethod", "sum", "super", "vars", "zip", "__import__",
		"breakpoint", "callable", "classmethod", "compile", "complex",
		"delattr", "eval", "exec", "memoryview", "property",
	),
	"typescript": setOf(
		"console", "JSON", "Math", "Object", "Array", "String", "Number",
		"Boolean", "Promise", "parseInt", "parseFloat", "isNaN", "isFinite",
		"decodeURI", "encodeURI", "setTimeout", "setInterval", "clearTimeout",
		"clearInterval", "fetch", "Symbol", "Map", "Set", "WeakMap", "WeakSet",
		"Proxy", "Reflect", "Error", "TypeError", "ReferenceError", "SyntaxError",
		"undefined", "null", "Infinity", "NaN", "eval", "globalThis",
	),
	"javascript": setOf(
		"console", "JSON", "Math", "Object", "Array", "String", "Number",
		"Boolean", "Promise", "parseInt", "parseFloat", "isNaN", "isFinite",
		"decodeURI", "encodeURI", "setTimeout", "setInterval", "clearTimeout",
		"clearInterval", "fetch", "Symbol", "Map", "Set", "WeakMap", "WeakSet",
		"Proxy", "Reflect", "Error", "TypeError", "ReferenceError", "SyntaxError",
		"undefined", "null", "Infinity", "NaN", "eval", "globalThis",
	),
	"rust": setOf(
		"println!", "print!", "eprintln!", "eprint!", "panic!", "assert!",
		"assert_eq!", "assert_ne!", "vec!", "format!", "write!", "writeln!",
		"todo!", "unimplemented!", "unreachable!", "dbg!", "include!",
		"include_str!", "include_bytes!", "concat!", "env!", "option_env!",
		"Box", "Vec", "String", "Option", "Result", "Ok", "Err", "Some", "None",
		"Drop", "Clone", "Copy", "Send", "Sync", "Sized", "Default",
	),
	"java": setOf(
		"System", "Object", "String", "Integer", "Double", "Boolean",
		"Math", "Arrays", "Collections",
	),
	"groovy": setOf(
		"System", "Object", "String", "Integer", "Double", "Boolean",
		"Math", "Arrays", "Collections",
	),
	"ruby": setOf(
		"puts", "print", "p", "pp", "gets", "raise", "require_relative",
		"attr_accessor", "attr_reader", "attr_writer", "include", "extend",
		"prepend", "private", "protected", "public", "lambda", "proc",
		"method", "send", "nil?", "is_a?", "kind_of?", "freeze",
		"frozen?", "dup", "clone",
	),
}

func setOf(names ...string) map[string]bool {
	s := make(map[string]bool, len(names))
	for _, n := range names {
		s[n] = true
	}
	return s
}

// IsBuiltin returns true if the symbol is a language builtin that should be dropped.
// It checks the full symbol, the base name (after last separator), and the qualifier (before first separator).
func IsBuiltin(language, symbol string) bool {
	set, ok := builtinSets[language]
	if !ok {
		return false
	}
	if set[symbol] {
		return true
	}
	// Check qualifier (e.g. "console" in "console.log")
	q := qualifierOf(symbol)
	if q != "" && set[q] {
		return true
	}
	// Check base name (e.g. "log" in "console.log") — less useful but included for completeness
	base := baseOf(symbol)
	return base != symbol && set[base]
}

// baseOf extracts the last segment of a qualified name (after . or ::).
func baseOf(symbol string) string {
	if i := strings.LastIndex(symbol, "."); i >= 0 {
		return symbol[i+1:]
	}
	if i := strings.LastIndex(symbol, "::"); i >= 0 {
		return symbol[i+2:]
	}
	return symbol
}

// qualifierOf extracts the first qualifier segment (before . or ::).
func qualifierOf(symbol string) string {
	if i := strings.Index(symbol, "::"); i >= 0 {
		return symbol[:i]
	}
	if i := strings.Index(symbol, "."); i >= 0 {
		return symbol[:i]
	}
	return ""
}
