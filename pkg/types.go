package pkg

// Node represents a single symbol in the repository.
// Slimmed to exactly what V1 needs — no more, no less.
type BeakonNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`       // function | method | class | module | variable
	Name       string `json:"name"`
	Language   string `json:"language"`
	FilePath   string `json:"file_path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Parent     string `json:"parent,omitempty"` // e.g. "AuthService" for AuthService.login
	SourceHash string `json:"source_hash"`      // sha1 of file — for incremental indexing
}

// CallEdge represents a directed call relationship between two symbols.
// Enrichment fields are populated at index time for external callees only.
type CallEdge struct {
	From string `json:"from"` // caller symbol name
	To   string `json:"to"`   // callee symbol name

	// Enrichment — set for external callees; empty for internal calls
	Package    string `json:"package,omitempty"`
	Stdlib     string `json:"stdlib,omitempty"`    // "yes" | "no" | "unknown"
	DevOnly    *bool  `json:"dev_only,omitempty"`
	Version    string `json:"version,omitempty"`
	Resolution string `json:"resolution,omitempty"` // "resolved" | "unresolved"
	Reason     string `json:"reason,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

// ExternalCallee holds enrichment data for a single external callee symbol.
type ExternalCallee struct {
	Package    string `json:"package,omitempty"`
	Stdlib     string `json:"stdlib,omitempty"`
	DevOnly    *bool  `json:"dev_only,omitempty"`
	Version    string `json:"version,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

// ExternalIndex maps callee symbol name → enrichment data.
// Stored in .beakon/graph/external.json at index time; used at query time.
type ExternalIndex map[string]ExternalCallee

// FileIndex is stored per source file in .beakon/nodes/*.json
type FileIndex struct {
	File    string          `json:"file"`
	Hash    string          `json:"hash"`
	Symbols []BeakonNode `json:"symbols"`
	Calls   []CallEdge      `json:"calls"`
}

// TraceStep is one node in a rich trace — symbol + location + source snippet.
type TraceStep struct {
	Symbol  string `json:"symbol"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	EndLine int    `json:"end_line,omitempty"`
	Snippet string `json:"snippet,omitempty"` // first 6 lines of the function
	Depth   int    `json:"depth"`
}

// NodeID generates a deterministic stable ID.
// Format: <language>:<kind>:<filepath>:<symbol>
// Example: go:function:auth/service.go:login
func NodeID(language, kind, filePath, symbol string) string {
	return language + ":" + kind + ":" + filePath + ":" + symbol
}
