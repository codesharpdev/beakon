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

	ignored := loadGitignore(root)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] || strings.HasPrefix(info.Name(), ".") || ignored[info.Name()] {
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

// loadGitignore reads root/.gitignore and returns a set of directory/file patterns to skip.
// Only simple patterns are supported (no glob negation, no complex rules).
func loadGitignore(root string) map[string]bool {
	ignored := make(map[string]bool)
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return ignored // no .gitignore — that's fine
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Normalize: strip trailing slash (we match dir names)
		line = strings.TrimSuffix(line, "/")
		ignored[line] = true
	}
	return ignored
}
