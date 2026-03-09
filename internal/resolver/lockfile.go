package resolver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// lockfileReader walks up from the source file's directory looking for lockfiles.
type lockfileReader struct {
	versions map[string]string // package name → version
	devOnly  map[string]bool   // packages that appear only in devDependencies (TS/JS)
	hasPkgJS bool              // whether a package.json was found (for dev_only nil semantics)
}

// newLockfileReader creates a reader for the given source file.
// It walks up from the file's directory to root looking for supported lockfiles.
func newLockfileReader(root, filePath, language string) *lockfileReader {
	lr := &lockfileReader{
		versions: map[string]string{},
		devOnly:  map[string]bool{},
	}
	absFile := filepath.Join(root, filePath)
	dir := filepath.Dir(absFile)

	// Walk up from file directory to root
	for {
		lr.tryLoad(dir, language)
		if dir == root || dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return lr
}

// Version returns the pinned version for a package, or "" if not found.
func (lr *lockfileReader) Version(pkg string) string {
	return lr.versions[pkg]
}

// DevOnly returns a *bool indicating whether pkg is a dev dependency.
// Returns nil if no package.json was found (dev_only unknown).
// Returns &true if the package is devDependencies-only.
// Returns &false if the package is in regular dependencies (or both).
func (lr *lockfileReader) DevOnly(pkg string) *bool {
	if !lr.hasPkgJS {
		return nil
	}
	b := lr.devOnly[pkg]
	return &b
}

func (lr *lockfileReader) tryLoad(dir, language string) {
	switch language {
	case "go":
		lr.loadGoMod(filepath.Join(dir, "go.mod"))
	case "typescript", "javascript":
		lr.loadPackageJSON(filepath.Join(dir, "package.json"))
		lr.loadPackageLock(filepath.Join(dir, "package-lock.json"))
	case "python":
		lr.loadRequirementsTxt(filepath.Join(dir, "requirements.txt"))
		lr.loadPoetryLock(filepath.Join(dir, "poetry.lock"))
	case "rust":
		lr.loadCargoToml(filepath.Join(dir, "Cargo.toml"))
	case "java", "groovy":
		// pom.xml and build.gradle are complex — skip version lookup for now
	case "ruby":
		lr.loadGemfileLock(filepath.Join(dir, "Gemfile.lock"))
	}
}

func (lr *lockfileReader) loadGoMod(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inRequire := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if inRequire || strings.HasPrefix(line, "require ") {
			raw := line
			if strings.HasPrefix(raw, "require ") {
				raw = strings.TrimPrefix(raw, "require ")
			}
			// Strip inline comment
			if i := strings.Index(raw, "//"); i >= 0 {
				raw = strings.TrimSpace(raw[:i])
			}
			parts := strings.Fields(raw)
			if len(parts) == 2 {
				// parts[0] = module path, parts[1] = version
				if _, ok := lr.versions[parts[0]]; !ok {
					lr.versions[parts[0]] = parts[1]
				}
			}
		}
	}
}

func (lr *lockfileReader) loadPackageJSON(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var pj struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pj); err != nil {
		return
	}
	lr.hasPkgJS = true
	// Load regular deps first — presence here means dev_only: false
	for pkg, ver := range pj.Dependencies {
		if _, ok := lr.versions[pkg]; !ok {
			lr.versions[pkg] = ver
		}
		lr.devOnly[pkg] = false // explicitly in prod deps
	}
	// Load dev deps — only mark devOnly if NOT already in prod deps
	for pkg, ver := range pj.DevDependencies {
		if _, ok := lr.versions[pkg]; !ok {
			lr.versions[pkg] = ver
		}
		if _, inProd := pj.Dependencies[pkg]; !inProd {
			lr.devOnly[pkg] = true
		}
	}
}

func (lr *lockfileReader) loadPackageLock(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return
	}
	for key, pkg := range lock.Packages {
		if pkg.Version == "" {
			continue
		}
		// key is like "node_modules/axios" → pkg name is "axios"
		name := strings.TrimPrefix(key, "node_modules/")
		if name != "" && name != key {
			if _, ok := lr.versions[name]; !ok {
				lr.versions[name] = pkg.Version
			}
		}
	}
}

func (lr *lockfileReader) loadRequirementsTxt(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// requests==2.28.0 or requests>=2.0
		for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">"} {
			if i := strings.Index(line, sep); i >= 0 {
				pkg := strings.TrimSpace(line[:i])
				ver := strings.TrimSpace(line[i+len(sep):])
				if pkg != "" {
					if _, ok := lr.versions[pkg]; !ok {
						lr.versions[pkg] = ver
					}
				}
				break
			}
		}
	}
}

func (lr *lockfileReader) loadPoetryLock(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Simple TOML-style parsing for [[package]] sections
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var name, version string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[[package]]" {
			name, version = "", ""
			continue
		}
		if strings.HasPrefix(line, "name = ") {
			name = unquote(strings.TrimPrefix(line, "name = "))
		} else if strings.HasPrefix(line, "version = ") {
			version = unquote(strings.TrimPrefix(line, "version = "))
		}
		if name != "" && version != "" {
			if _, ok := lr.versions[name]; !ok {
				lr.versions[name] = version
			}
			name, version = "", ""
		}
	}
}

func (lr *lockfileReader) loadCargoToml(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inDeps := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[dependencies]" || line == "[dev-dependencies]" || line == "[build-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}
		if !inDeps || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// serde = "1.0" or serde = { version = "1.0", features = [...] }
		if eqIdx := strings.Index(line, "="); eqIdx >= 0 {
			pkg := strings.TrimSpace(line[:eqIdx])
			rest := strings.TrimSpace(line[eqIdx+1:])
			var ver string
			if strings.HasPrefix(rest, "\"") {
				ver = unquote(rest)
			} else if strings.Contains(rest, "version") {
				// { version = "1.0", ... }
				if vIdx := strings.Index(rest, "version"); vIdx >= 0 {
					after := rest[vIdx+7:]
					if eIdx := strings.Index(after, "="); eIdx >= 0 {
						ver = unquote(strings.TrimSpace(after[eIdx+1:]))
						if commaIdx := strings.Index(ver, ","); commaIdx >= 0 {
							ver = strings.TrimSpace(ver[:commaIdx])
						}
						ver = strings.Trim(ver, "}")
					}
				}
			}
			if pkg != "" && ver != "" {
				if _, ok := lr.versions[pkg]; !ok {
					lr.versions[pkg] = ver
				}
			}
		}
	}
}

func (lr *lockfileReader) loadGemfileLock(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inGems := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "GEM" || strings.HasPrefix(line, "GEM\n") {
			inGems = true
			continue
		}
		if strings.HasPrefix(line, "  specs:") {
			continue
		}
		if inGems && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") {
			// "    gemname (version)"
			trimmed := strings.TrimSpace(line)
			if paren := strings.Index(trimmed, " ("); paren >= 0 {
				gem := trimmed[:paren]
				ver := strings.Trim(trimmed[paren+2:], ")")
				if _, ok := lr.versions[gem]; !ok {
					lr.versions[gem] = ver
				}
			}
		}
		if inGems && line == "" {
			inGems = false
		}
	}
}
