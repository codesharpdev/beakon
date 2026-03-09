// Package resolver provides builtin filtering and external callee enrichment
// for all languages supported by Beakon.
//
// Enrichment happens at index time: builtins are dropped, external callees
// are matched against import statements and annotated with package metadata
// and version information from lockfiles.
package resolver

import (
	"github.com/beakon/beakon/pkg"
)

// Enrich filters builtins from the call edges and enriches external callees
// with package metadata. It returns a new slice — the original is not modified.
//
// root is the repository root (for lockfile lookup).
// filePath is the source file path relative to root.
func Enrich(root, filePath, language string, src []byte, calls []pkg.CallEdge) []pkg.CallEdge {
	imports := parseImports(language, src)
	lf := newLockfileReader(root, filePath, language)

	result := make([]pkg.CallEdge, 0, len(calls))
	for _, edge := range calls {
		enriched, keep := enrichEdge(edge, language, imports, lf)
		if keep {
			result = append(result, enriched)
		}
	}
	return result
}

// enrichEdge returns the enriched edge and whether to keep it.
// Builtins return (edge, false) — they should be dropped.
func enrichEdge(edge pkg.CallEdge, language string, imports importMap, lf *lockfileReader) (pkg.CallEdge, bool) {
	callee := edge.To

	// 1. Drop builtins — they are not external APIs
	if IsBuiltin(language, callee) {
		return edge, false
	}

	// 2. Try to resolve via qualifier
	qualifier := qualifierOf(callee)
	if qualifier != "" {
		if imp, ok := imports[qualifier]; ok {
			edge.Package = imp.Package
			edge.Stdlib = imp.Stdlib
			edge.Resolution = imp.Resolution
			edge.Reason = imp.Reason
			if imp.Resolution == "resolved" && imp.Stdlib != "yes" {
				edge.Version = lf.Version(imp.Package)
				edge.DevOnly = lf.DevOnly(imp.Package)
			}
			return edge, true
		}
	}

	// 3. Try direct binding (the callee itself is the import key — e.g. Python/TS named imports)
	if imp, ok := imports[callee]; ok {
		edge.Package = imp.Package
		edge.Stdlib = imp.Stdlib
		edge.Resolution = imp.Resolution
		edge.Reason = imp.Reason
		if imp.Resolution == "resolved" && imp.Stdlib != "yes" {
			edge.Version = lf.Version(imp.Package)
			edge.DevOnly = lf.DevOnly(imp.Package)
		}
		return edge, true
	}

	// 4. No import match — keep the edge as-is (may be an internal call resolved at query time)
	return edge, true
}
