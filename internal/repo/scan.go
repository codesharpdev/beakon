package repo

import (
	"os"
	"path/filepath"
	"strings"
)

var skipDirs = map[string]bool{
	".git":         true,
	".beakon":      true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

var langByExt = map[string]string{
	".go":  "go",
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "javascript",
	".jsx": "javascript",
	".py":  "python",
}

// SourceFile is a discovered file with its detected language.
type SourceFile struct {
	Path     string
	Language string
}

// Scan walks root and returns all supported source files.
func Scan(root string) ([]SourceFile, error) {
	var files []SourceFile

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := langByExt[ext]; ok {
			rel, _ := filepath.Rel(root, path)
			files = append(files, SourceFile{Path: rel, Language: lang})
		}
		return nil
	})

	return files, err
}
