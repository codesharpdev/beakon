package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/beakon/beakon/internal/graph"
	"github.com/beakon/beakon/internal/index"
	"github.com/beakon/beakon/internal/resolver"
	"github.com/beakon/beakon/internal/symbols"
	"github.com/beakon/beakon/pkg"
)

// globalMu serializes all writes to symbols.json, map.json, and graph files.
// Multiple file updates must not rebuild global indexes concurrently.
var globalMu sync.Mutex

// UpdateResult summarizes a single-file incremental update.
type UpdateResult struct {
	FilePath       string
	SymbolsBefore  int
	SymbolsAfter   int
	Duration       time.Duration
	Skipped        bool   // true if hash unchanged
	SkipReason     string
}

// UpdateFile performs a surgical update for a single changed file.
// Strategy B: load all existing data, remove old file's contribution,
// add new file's contribution, rewrite affected index files.
func UpdateFile(root, filePath string) (*UpdateResult, error) {
	start := time.Now()

	// Normalize to relative path
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		rel = filePath
	}

	absPath := filepath.Join(root, rel)

	// File deleted — remove from index
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return removeFile(root, rel, start)
	}

	// Compute new hash
	newHash := symbols.HashFile(absPath)

	// Check if file actually changed
	if !index.NeedsUpdate(root, rel, newHash) {
		return &UpdateResult{
			FilePath:   rel,
			Skipped:    true,
			SkipReason: "hash unchanged",
			Duration:   time.Since(start),
		}, nil
	}

	// Detect language
	lang, ok := detectLang(rel)
	if !ok {
		return &UpdateResult{
			FilePath:   rel,
			Skipped:    true,
			SkipReason: "unsupported language",
			Duration:   time.Since(start),
		}, nil
	}

	// Read new source and extract symbols
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	newSyms, newCalls := symbols.Extract(rel, lang, src)
	newCalls = resolver.Enrich(root, rel, lang, src, newCalls)

	// Count old symbols for result
	oldCount := 0
	if oldFI, err := index.Read(root, rel); err == nil {
		oldCount = len(oldFI.Symbols)
	}

	// Write new file index
	fi := pkg.FileIndex{
		File:    rel,
		Hash:    newHash,
		Symbols: newSyms,
		Calls:   newCalls,
	}
	if err := index.Write(root, fi); err != nil {
		return nil, fmt.Errorf("write file index: %w", err)
	}

	// Rebuild global indexes surgically
	globalMu.Lock()
	err = rebuildGlobal(root, rel, newSyms, newCalls)
	globalMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("rebuild global: %w", err)
	}

	return &UpdateResult{
		FilePath:      rel,
		SymbolsBefore: oldCount,
		SymbolsAfter:  len(newSyms),
		Duration:      time.Since(start),
	}, nil
}

// removeFile removes a deleted file's contribution from the index.
func removeFile(root, rel string, start time.Time) (*UpdateResult, error) {
	oldCount := 0
	if oldFI, err := index.Read(root, rel); err == nil {
		oldCount = len(oldFI.Symbols)
	}

	// Remove the file index
	index.DeleteFile(root, rel)

	// Rebuild global without this file
	globalMu.Lock()
	err := rebuildGlobal(root, rel, nil, nil)
	globalMu.Unlock()
	if err != nil {
		return nil, err
	}

	return &UpdateResult{
		FilePath:      rel,
		SymbolsBefore: oldCount,
		SymbolsAfter:  0,
		Duration:      time.Since(start),
	}, nil
}

// rebuildGlobal reloads all file indexes, replaces the changed file's
// contribution, and rewrites symbols.json, map.json, and graph files.
// This is Strategy B — surgical O(files) rebuild, not O(repo).
func rebuildGlobal(root, changedFile string, newSyms []pkg.BeakonNode, newCalls []pkg.CallEdge) error {
	// Load all existing file indexes
	allFiles, err := index.ReadAll(root)
	if err != nil {
		return fmt.Errorf("read all: %w", err)
	}

	var allSymbols []pkg.BeakonNode
	var allEdges []pkg.CallEdge

	for _, fi := range allFiles {
		// Skip the file we just updated (we'll add the new version)
		if fi.File == changedFile {
			continue
		}
		allSymbols = append(allSymbols, fi.Symbols...)
		allEdges = append(allEdges, fi.Calls...)
	}

	// Add the new version of the changed file
	allSymbols = append(allSymbols, newSyms...)
	allEdges = append(allEdges, newCalls...)

	// Drop edges where the callee no longer exists in the symbol table.
	// This handles cross-file invalidation: if a symbol is removed from one
	// file, other files that called it still have stale edges in their
	// FileIndex. Filter them out before rebuilding the graph.
	knownSymbols := make(map[string]bool, len(allSymbols))
	for _, s := range allSymbols {
		knownSymbols[s.Name] = true
	}
	filtered := allEdges[:0]
	for _, e := range allEdges {
		// Keep internal calls to known symbols, and enriched external calls
		if knownSymbols[e.To] || e.Package != "" {
			filtered = append(filtered, e)
		}
	}
	allEdges = filtered

	// Rewrite all global index files
	callsFrom, callsTo := graph.Build(allEdges)
	if err := graph.Write(root, callsFrom, callsTo); err != nil {
		return err
	}
	extIdx := graph.BuildExternal(allEdges)
	if err := graph.WriteExternal(root, extIdx); err != nil {
		return err
	}
	if err := index.WriteSymbols(root, allSymbols); err != nil {
		return err
	}
	return index.WriteMap(root, BuildMap(allSymbols))
}

func detectLang(filePath string) (string, bool) {
	switch {
	case hasSuffix(filePath, ".go"):
		return "go", true
	case hasSuffix(filePath, ".ts"), hasSuffix(filePath, ".tsx"):
		return "typescript", true
	case hasSuffix(filePath, ".js"), hasSuffix(filePath, ".jsx"):
		return "javascript", true
	case hasSuffix(filePath, ".py"):
		return "python", true
	}
	return "", false
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
