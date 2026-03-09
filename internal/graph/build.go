package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/beakon/beakon/pkg"
)

const graphDir = ".codeindex/graph"

// CallsFrom maps symbol → []symbols it calls
// Stored in .codeindex/graph/calls_from.json
type CallsFrom map[string][]string

// CallsTo maps symbol → []symbols that call it
// Stored in .codeindex/graph/calls_to.json
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

// Write persists both graph files to .codeindex/graph/
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

// TraceRich performs BFS and enriches each step with file location and code snippet.
// symIndex is a map of symbol name → CodeIndexNode for fast lookup.
// repoRoot is needed to read source files live.
func TraceRich(symbol string, from CallsFrom, symIndex map[string]pkg.CodeIndexNode, repoRoot string) []pkg.TraceStep {
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
