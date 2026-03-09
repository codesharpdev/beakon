package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/codeindex/codeindex/internal/graph"
	"github.com/codeindex/codeindex/internal/index"
	"github.com/codeindex/codeindex/internal/repo"
	"github.com/codeindex/codeindex/internal/symbols"
	"github.com/codeindex/codeindex/pkg"
)

// Result summarizes a completed indexing run.
type Result struct {
	FilesIndexed int
	FilesSkipped int
	SymbolsFound int
	Duration     time.Duration
	Errors       []string
}

// Run performs a full index of the repository at root.
// Files whose hash hasn't changed are skipped (incremental).
func Run(root string) (*Result, error) {
	start := time.Now()

	if err := index.Init(root); err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}

	files, err := repo.Scan(root)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	var mu sync.Mutex
	var allSymbols []pkg.CodeIndexNode
	var allEdges []pkg.CallEdge
	var errors []string
	indexed, skipped := 0, 0

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for _, f := range files {
		wg.Add(1)
		go func(sf repo.SourceFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			absPath := filepath.Join(root, sf.Path)
			src, err := os.ReadFile(absPath)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("read %s: %v", sf.Path, err))
				mu.Unlock()
				return
			}

			hash := symbols.HashFile(absPath)

			// Skip files that haven't changed
			if !index.NeedsUpdate(root, sf.Path, hash) {
				// Still need to load existing symbols for graph rebuild
				fi, err := index.Read(root, sf.Path)
				if err == nil {
					mu.Lock()
					allSymbols = append(allSymbols, fi.Symbols...)
					allEdges = append(allEdges, fi.Calls...)
					skipped++
					mu.Unlock()
				}
				return
			}

			syms, calls := symbols.Extract(sf.Path, sf.Language, src)
			fi := pkg.FileIndex{
				File:    sf.Path,
				Hash:    hash,
				Symbols: syms,
				Calls:   calls,
			}

			if err := index.Write(root, fi); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("write %s: %v", sf.Path, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			allSymbols = append(allSymbols, syms...)
			allEdges = append(allEdges, calls...)
			indexed++
			mu.Unlock()
		}(f)
	}

	wg.Wait()

	if err := writeIndexFiles(root, allSymbols, allEdges); err != nil {
		return nil, err
	}

	index.WriteMeta(root, index.Meta{
		Version:   "1",
		IndexedAt: time.Now(),
		RepoRoot:  root,
		FileCount: indexed + skipped,
		SymCount:  len(allSymbols),
	})

	return &Result{
		FilesIndexed: indexed,
		FilesSkipped: skipped,
		SymbolsFound: len(allSymbols),
		Duration:     time.Since(start),
		Errors:       errors,
	}, nil
}

// writeIndexFiles rebuilds symbols.json, map.json, and graph files.
func writeIndexFiles(root string, allSymbols []pkg.CodeIndexNode, allEdges []pkg.CallEdge) error {
	callsFrom, callsTo := graph.Build(allEdges)
	if err := graph.Write(root, callsFrom, callsTo); err != nil {
		return fmt.Errorf("write graph: %w", err)
	}
	if err := index.WriteSymbols(root, allSymbols); err != nil {
		return fmt.Errorf("write symbols: %w", err)
	}
	if err := index.WriteMap(root, BuildMap(allSymbols)); err != nil {
		return fmt.Errorf("write map: %w", err)
	}
	return nil
}

// BuildMap groups symbol names by directory.
func BuildMap(syms []pkg.CodeIndexNode) index.MapIndex {
	m := make(index.MapIndex)
	for _, s := range syms {
		parts := strings.Split(s.FilePath, "/")
		dir := "."
		if len(parts) > 1 {
			dir = strings.Join(parts[:len(parts)-1], "/")
		}
		m[dir] = append(m[dir], s.Name)
	}
	return m
}
