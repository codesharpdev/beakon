package context

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/beakon/beakon/internal/graph"
	"github.com/beakon/beakon/internal/index"
	"github.com/beakon/beakon/pkg"
)

// CodeBlock is a symbol with its live source code attached.
type CodeBlock struct {
	Symbol string `json:"symbol"`
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Code   string `json:"code"`
}

// Bundle is the complete context package sent to the LLM.
type Bundle struct {
	Query          string      `json:"query"`
	Anchor         CodeBlock   `json:"anchor"`
	Callers        []CodeBlock `json:"callers"`
	Callees        []CodeBlock `json:"callees"`
	Files          []string    `json:"files"`
	TokenEstimate  int         `json:"token_estimate"`
}

// Engine assembles context bundles from the index.
type Engine struct {
	root    string
	symIdx  map[string]pkg.CodeIndexNode
	from    graph.CallsFrom
	to      graph.CallsTo
	loaded  bool
}

// NewEngine creates a context engine for the repository at root.
func NewEngine(root string) *Engine {
	return &Engine{root: root}
}

// load initialises the in-memory index once.
func (e *Engine) load() error {
	if e.loaded {
		return nil
	}

	syms, err := index.ReadSymbols(e.root)
	if err != nil {
		return err
	}
	e.symIdx = make(map[string]pkg.CodeIndexNode, len(syms))
	for _, s := range syms {
		e.symIdx[s.Name] = s
	}

	e.from, err = graph.ReadFrom(e.root)
	if err != nil {
		return err
	}
	e.to, err = graph.ReadTo(e.root)
	if err != nil {
		return err
	}

	e.loaded = true
	return nil
}

// Assemble builds a complete context bundle for the given symbol.
func (e *Engine) Assemble(query string) (*Bundle, error) {
	if err := e.load(); err != nil {
		return nil, err
	}

	// Find the anchor symbol
	anchor, ok := e.findSymbol(query)
	if !ok {
		return nil, &SymbolNotFound{Name: query}
	}

	bundle := &Bundle{
		Query:  query,
		Anchor: e.toBlock(anchor),
	}

	// Direct callees — what this symbol calls
	for _, calleeName := range e.from[anchor.Name] {
		if sym, ok := e.findSymbol(calleeName); ok {
			bundle.Callees = append(bundle.Callees, e.toBlock(sym))
		} else {
			// Symbol not in index (external lib etc) — include name only
			bundle.Callees = append(bundle.Callees, CodeBlock{
				Symbol: calleeName,
				Kind:   "external",
			})
		}
	}

	// Direct callers — what calls this symbol
	for _, callerName := range e.to[anchor.Name] {
		if sym, ok := e.findSymbol(callerName); ok {
			bundle.Callers = append(bundle.Callers, e.toBlock(sym))
		}
	}

	// Collect unique files involved
	bundle.Files = uniqueFiles(bundle)

	// Estimate tokens (chars / 4 is the standard approximation)
	bundle.TokenEstimate = estimateTokens(bundle)

	return bundle, nil
}

// toBlock converts a CodeIndexNode to a CodeBlock with live source.
func (e *Engine) toBlock(node pkg.CodeIndexNode) CodeBlock {
	block := CodeBlock{
		Symbol: node.Name,
		Kind:   node.Kind,
		File:   node.FilePath,
		Start:  node.StartLine,
		End:    node.EndLine,
	}
	block.Code = fetchCode(e.root, node.FilePath, node.StartLine, node.EndLine)
	return block
}

// findSymbol looks up a symbol by exact name or partial suffix match.
func (e *Engine) findSymbol(name string) (pkg.CodeIndexNode, bool) {
	// Exact match first
	if n, ok := e.symIdx[name]; ok {
		return n, true
	}
	// Partial match: "login" matches "AuthService.login"
	nameLower := strings.ToLower(name)
	for k, n := range e.symIdx {
		if strings.ToLower(k) == nameLower {
			return n, true
		}
		if strings.HasSuffix(strings.ToLower(k), "."+nameLower) {
			return n, true
		}
	}
	return pkg.CodeIndexNode{}, false
}

// fetchCode reads the source lines for a symbol live from disk.
func fetchCode(root, filePath string, start, end int) string {
	if filePath == "" || start == 0 {
		return ""
	}
	absPath := filepath.Join(root, filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// uniqueFiles returns deduplicated list of files across the whole bundle.
func uniqueFiles(b *Bundle) []string {
	seen := map[string]bool{}
	var files []string

	add := func(f string) {
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}

	add(b.Anchor.File)
	for _, c := range b.Callees {
		add(c.File)
	}
	for _, c := range b.Callers {
		add(c.File)
	}
	return files
}

// estimateTokens gives a rough token count for the bundle (chars / 4).
func estimateTokens(b *Bundle) int {
	total := len(b.Anchor.Code)
	for _, c := range b.Callees {
		total += len(c.Code)
	}
	for _, c := range b.Callers {
		total += len(c.Code)
	}
	return total / 4
}

// SymbolNotFound is returned when a symbol cannot be located in the index.
type SymbolNotFound struct {
	Name string
}

func (e *SymbolNotFound) Error() string {
	return "symbol not found: " + e.Name + " — run 'codeindex index' first"
}
