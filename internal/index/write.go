package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/beakon/beakon/pkg"
)

const (
	beakonDir = ".beakon"
	nodesDir  = ".beakon/nodes"
)

// Meta holds top-level index metadata.
type Meta struct {
	Version   string    `json:"version"`
	IndexedAt time.Time `json:"indexed_at"`
	RepoRoot  string    `json:"repo_root"`
	FileCount int       `json:"file_count"`
	SymCount  int       `json:"sym_count"`
}

// SymbolsIndex is the flat list of all symbols — stored in symbols.json
type SymbolsIndex struct {
	Symbols []pkg.CodeIndexNode `json:"symbols"`
}

// MapIndex is the architectural overview — stored in map.json
type MapIndex map[string][]string // dir → []symbol names

// Write persists a FileIndex for one source file.
func Write(root string, fi pkg.FileIndex) error {
	dir := filepath.Join(root, nodesDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	key := fileKey(fi.File)
	return writeJSON(filepath.Join(dir, key+".json"), fi)
}

// Read loads the FileIndex for a single source file.
func Read(root, filePath string) (*pkg.FileIndex, error) {
	path := filepath.Join(root, nodesDir, fileKey(filePath)+".json")
	var fi pkg.FileIndex
	if err := readJSON(path, &fi); err != nil {
		return nil, err
	}
	return &fi, nil
}

// ReadAll loads every FileIndex in .codeindex/files/
func ReadAll(root string) ([]pkg.FileIndex, error) {
	dir := filepath.Join(root, nodesDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var mu sync.Mutex
	var all []pkg.FileIndex
	var wg sync.WaitGroup

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			var fi pkg.FileIndex
			if err := readJSON(filepath.Join(dir, name), &fi); err == nil {
				mu.Lock()
				all = append(all, fi)
				mu.Unlock()
			}
		}(e.Name())
	}

	wg.Wait()
	return all, nil
}

// WriteSymbols writes the flat symbols.json index.
func WriteSymbols(root string, symbols []pkg.CodeIndexNode) error {
	path := filepath.Join(root, beakonDir, "symbols.json")
	return writeJSON(path, SymbolsIndex{Symbols: symbols})
}

// ReadSymbols loads symbols.json.
func ReadSymbols(root string) ([]pkg.CodeIndexNode, error) {
	path := filepath.Join(root, beakonDir, "symbols.json")
	var si SymbolsIndex
	if err := readJSON(path, &si); err != nil {
		return nil, err
	}
	return si.Symbols, nil
}

// WriteMap writes map.json.
func WriteMap(root string, m MapIndex) error {
	path := filepath.Join(root, beakonDir, "map.json")
	return writeJSON(path, m)
}

// ReadMap loads map.json.
func ReadMap(root string) (MapIndex, error) {
	path := filepath.Join(root, beakonDir, "map.json")
	var m MapIndex
	err := readJSON(path, &m)
	return m, err
}

// WriteMeta writes meta.json.
func WriteMeta(root string, meta Meta) error {
	path := filepath.Join(root, beakonDir, "meta.json")
	return writeJSON(path, meta)
}

// ReadMeta loads meta.json.
func ReadMeta(root string) (Meta, error) {
	path := filepath.Join(root, beakonDir, "meta.json")
	var m Meta
	err := readJSON(path, &m)
	return m, err
}

// NeedsUpdate returns true if the file's hash differs from stored hash.
func NeedsUpdate(root, filePath, currentHash string) bool {
	fi, err := Read(root, filePath)
	if err != nil {
		return true // not yet indexed
	}
	return fi.Hash != currentHash
}

// Init creates the .codeindex directory structure.
func Init(root string) error {
	dirs := []string{
		filepath.Join(root, beakonDir),
		filepath.Join(root, nodesDir),
		filepath.Join(root, ".beakon/graph"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

// fileKey converts a file path to a flat storage key.
// auth/service.go → auth_service.go
func fileKey(filePath string) string {
	key := strings.ReplaceAll(filePath, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	return key
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

// DeleteFile removes a file's index entry from .codeindex/files/
func DeleteFile(root, filePath string) error {
	path := filepath.Join(root, nodesDir, fileKey(filePath)+".json")
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // already gone
	}
	return err
}
