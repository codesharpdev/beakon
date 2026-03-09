package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codesharpdev/beakon/pkg"
)

const graphDir = ".beakon/graph"

// CallsFrom maps symbol → []symbols it calls
// Stored in .beakon/graph/calls_from.json
type CallsFrom map[string][]string

// CallsTo maps symbol → []symbols that call it
// Stored in .beakon/graph/calls_to.json
type CallsTo map[string][]string

// Build constructs both directions of the call graph from all edges.
func Build(edges []pkg.CallEdge) (CallsFrom, CallsTo) {
	from := make(CallsFrom)
	to := make(CallsTo)

	for _, e := range edges {
		// Deduplicate
		if !contains(from[e.From], e.To) {
			from[e.From] = append(from[e.From], e.To)
		}
		if !contains(to[e.To], e.From) {
			to[e.To] = append(to[e.To], e.From)
		}
	}

	return from, to
}

// Write persists both graph files to .beakon/graph/
func Write(root string, from CallsFrom, to CallsTo) error {
	dir := filepath.Join(root, graphDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "calls_from.json"), from); err != nil {
		return fmt.Errorf("calls_from: %w", err)
	}
	if err := writeJSON(filepath.Join(dir, "calls_to.json"), to); err != nil {
		return fmt.Errorf("calls_to: %w", err)
	}
	return nil
}

// BuildExternal collects enrichment data from enriched call edges into an ExternalIndex.
// Only edges that have a Package set are included.
// When two edges resolve the same callee, "resolved" beats "unresolved".
func BuildExternal(edges []pkg.CallEdge) pkg.ExternalIndex {
	idx := make(pkg.ExternalIndex)
	for _, e := range edges {
		if e.Package == "" {
			continue
		}
		candidate := pkg.ExternalCallee{
			Package:    e.Package,
			Stdlib:     e.Stdlib,
			DevOnly:    e.DevOnly,
			Version:    e.Version,
			Resolution: e.Resolution,
			Reason:     e.Reason,
			Hint:       e.Hint,
		}
		existing, exists := idx[e.To]
		if !exists {
			idx[e.To] = candidate
			continue
		}
		// Prefer resolved over unresolved; prefer richer version info
		if existing.Resolution != "resolved" && candidate.Resolution == "resolved" {
			idx[e.To] = candidate
		} else if existing.Version == "" && candidate.Version != "" {
			idx[e.To] = candidate
		}
	}
	return idx
}

// WriteExternal persists the ExternalIndex to .beakon/graph/external.json
func WriteExternal(root string, idx pkg.ExternalIndex) error {
	dir := filepath.Join(root, graphDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "external.json"), idx)
}

// ReadExternal loads .beakon/graph/external.json
func ReadExternal(root string) (pkg.ExternalIndex, error) {
	var idx pkg.ExternalIndex
	err := readJSON(filepath.Join(root, graphDir, "external.json"), &idx)
	if idx == nil {
		idx = make(pkg.ExternalIndex)
	}
	return idx, err
}

// ReadFrom loads calls_from.json
func ReadFrom(root string) (CallsFrom, error) {
	var m CallsFrom
	err := readJSON(filepath.Join(root, graphDir, "calls_from.json"), &m)
	if m == nil {
		m = make(CallsFrom)
	}
	return m, err
}

// ReadTo loads calls_to.json
func ReadTo(root string) (CallsTo, error) {
	var m CallsTo
	err := readJSON(filepath.Join(root, graphDir, "calls_to.json"), &m)
	if m == nil {
		m = make(CallsTo)
	}
	return m, err
}

// Trace performs BFS from a symbol through calls_from.
func Trace(symbol string, from CallsFrom) []string {
	queue := []string{symbol}
	visited := map[string]bool{}
	var result []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)
		for _, next := range from[current] {
			queue = append(queue, next)
		}
	}
	return result
}

// Impact performs reverse BFS from a symbol through calls_to.
// It returns every symbol that directly or indirectly depends on the given symbol.
// This is the answer to: "if I change X, what else might break?"
func Impact(symbol string, to CallsTo) []string {
	queue := []string{symbol}
	visited := map[string]bool{}
	var result []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)
		for _, caller := range to[current] {
			queue = append(queue, caller)
		}
	}
	return result
}

// TraceRich performs BFS and enriches each step with file location and code snippet.
// symIndex is a map of symbol name → Node for fast lookup.
// repoRoot is needed to read source files live.
func TraceRich(symbol string, from CallsFrom, symIndex map[string]pkg.BeakonNode, repoRoot string) []pkg.TraceStep {
	queue := []struct {
		name  string
		depth int
	}{{symbol, 0}}
	visited := map[string]bool{}
	var steps []pkg.TraceStep

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.name] {
			continue
		}
		visited[current.name] = true

		step := pkg.TraceStep{
			Symbol: current.name,
			Depth:  current.depth,
		}

		// Enrich with symbol metadata if available
		if node, ok := symIndex[current.name]; ok {
			step.File = node.FilePath
			step.Line = node.StartLine
			step.EndLine = node.EndLine
			step.Snippet = fetchSnippet(repoRoot, node.FilePath, node.StartLine, node.EndLine)
		} else {
			// Try partial name match (e.g. "login" matches "AuthService.login")
			for name, node := range symIndex {
				if hasSuffix(name, "."+current.name) || name == current.name {
					step.File = node.FilePath
					step.Line = node.StartLine
					step.EndLine = node.EndLine
					step.Snippet = fetchSnippet(repoRoot, node.FilePath, node.StartLine, node.EndLine)
					break
				}
			}
		}

		steps = append(steps, step)

		for _, next := range from[current.name] {
			queue = append(queue, struct {
				name  string
				depth int
			}{next, current.depth + 1})
		}
	}
	return steps
}

// ExplainResult is the output of the explain command.
type ExplainResult struct {
	Feature string          `json:"feature"`
	Entry   string          `json:"entry,omitempty"`
	Flow    []pkg.TraceStep `json:"flow"`
	Files   []string        `json:"files"`
}

// fetchSnippet reads lines from a source file, capped at 6 lines.
func fetchSnippet(repoRoot, filePath string, start, end int) string {
	if filePath == "" || start == 0 {
		return ""
	}
	absPath := filepath.Join(repoRoot, filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	lines := splitLines(string(data))
	if start < 1 {
		start = 1
	}
	if start > len(lines) {
		return ""
	}
	// Cap at 6 lines for inline display
	capEnd := start + 5
	if end < capEnd {
		capEnd = end
	}
	if capEnd > len(lines) {
		capEnd = len(lines)
	}
	result := ""
	for _, l := range lines[start-1 : capEnd] {
		result += l + "\n"
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}
