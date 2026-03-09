package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/beakon/beakon/internal/code"
	ctx "github.com/beakon/beakon/internal/context"
	"github.com/beakon/beakon/internal/graph"
	"github.com/beakon/beakon/internal/index"
	"github.com/beakon/beakon/internal/indexer"
	"github.com/beakon/beakon/pkg"
)

var human bool

var root = &cobra.Command{
	Use:   "beakon",
	Short: "Structural code intelligence for AI agents",
}

// ── index ─────────────────────────────────────────────────────────────────────

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		result, err := indexer.Run(repoRoot)
		if err != nil {
			return err
		}
		if human {
			fmt.Printf("✓ Indexed %d files (%d skipped), %d symbols in %s\n",
				result.FilesIndexed, result.FilesSkipped,
				result.SymbolsFound, result.Duration.Round(1000000))
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "  warn: %s\n", e)
			}
		} else {
			printJSON(map[string]any{
				"files_indexed": result.FilesIndexed,
				"files_skipped": result.FilesSkipped,
				"symbols":       result.SymbolsFound,
				"duration_ms":   result.Duration.Milliseconds(),
				"errors":        result.Errors,
			})
		}
		return nil
	},
}

// ── watch ─────────────────────────────────────────────────────────────────────

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch repository and incrementally update index on file changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()

		// Ensure index exists before watching
		if _, err := index.ReadMeta(repoRoot); err != nil {
			fmt.Println("No index found. Running initial index first...")
			result, err := indexer.Run(repoRoot)
			if err != nil {
				return fmt.Errorf("initial index: %w", err)
			}
			fmt.Printf("✓ Initial index: %d files, %d symbols\n",
				result.FilesIndexed, result.SymbolsFound)
		}

		w, err := indexer.NewWatcher(repoRoot)
		if err != nil {
			return fmt.Errorf("start watcher: %w", err)
		}
		defer w.Stop()

		fmt.Printf("watching %s — press Ctrl+C to stop\n", repoRoot)

		// Handle Ctrl+C gracefully
		go func() {
			ch := make(chan os.Signal, 1)
			// signal.Notify(ch, os.Interrupt) — wire this up properly
			<-ch
			w.Stop()
		}()

		// Start watching in background, consume events in foreground
		go w.Start()

		for event := range w.Events {
			if event.Err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", event.Err)
				continue
			}

			rel, _ := filepath.Rel(repoRoot, event.FilePath)
			r := event.Result

			if r.Skipped {
				// Don't print skipped events — too noisy
				continue
			}

			if human {
				fmt.Printf("↻ %s  %d→%d symbols  %s\n",
					rel, r.SymbolsBefore, r.SymbolsAfter,
					r.Duration.Round(1000000))
			} else {
				printJSON(map[string]any{
					"file":            rel,
					"symbols_before":  r.SymbolsBefore,
					"symbols_after":   r.SymbolsAfter,
					"duration_ms":     r.Duration.Milliseconds(),
				})
			}
		}
		return nil
	},
}

// ── map ───────────────────────────────────────────────────────────────────────

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Print architectural overview",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		m, err := index.ReadMap(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		if human {
			for dir, syms := range m {
				fmt.Printf("%s/\n", dir)
				for _, s := range syms {
					fmt.Printf("  %s\n", s)
				}
			}
		} else {
			printJSON(m)
		}
		return nil
	},
}

// ── trace ─────────────────────────────────────────────────────────────────────

var traceCmd = &cobra.Command{
	Use:   "trace <symbol>",
	Short: "Trace call chain with inline code snippets",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		callsFrom, err := graph.ReadFrom(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		if human {
			symIdx, _ := loadSymIndex(repoRoot)
			steps := graph.TraceRich(args[0], callsFrom, symIdx, repoRoot)
			printRichTrace(steps)
		} else {
			chain := graph.Trace(args[0], callsFrom)
			printJSON(map[string]any{"symbol": args[0], "chain": chain})
		}
		return nil
	},
}

// ── explain ───────────────────────────────────────────────────────────────────

var explainCmd = &cobra.Command{
	Use:   "explain <symbol>",
	Short: "Explain a feature: full flow, files involved",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		callsFrom, err := graph.ReadFrom(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		symIdx, _ := loadSymIndex(repoRoot)
		steps := graph.TraceRich(args[0], callsFrom, symIdx, repoRoot)

		seen := map[string]bool{}
		var files []string
		for _, s := range steps {
			if s.File != "" && !seen[s.File] {
				seen[s.File] = true
				files = append(files, s.File)
			}
		}

		result := graph.ExplainResult{Feature: args[0], Flow: steps, Files: files}

		if human {
			fmt.Printf("Feature: %s\n\n", result.Feature)
			fmt.Println("Flow:")
			for _, step := range result.Flow {
				indent := strings.Repeat("  ", step.Depth+1)
				if step.File != "" {
					fmt.Printf("%s%s  (%s:%d)\n", indent, step.Symbol, step.File, step.Line)
				} else {
					fmt.Printf("%s%s\n", indent, step.Symbol)
				}
			}
			fmt.Println("\nFiles involved:")
			for _, f := range result.Files {
				fmt.Printf("  %s\n", f)
			}
		} else {
			printJSON(result)
		}
		return nil
	},
}

// ── callers ───────────────────────────────────────────────────────────────────

var callersCmd = &cobra.Command{
	Use:   "callers <symbol>",
	Short: "Show all symbols that call this symbol",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		callsTo, err := graph.ReadTo(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		callers := callsTo[args[0]]
		if callers == nil {
			callers = []string{}
		}
		if human {
			fmt.Printf("callers of %s:\n", args[0])
			for _, c := range callers {
				fmt.Printf("  %s\n", c)
			}
		} else {
			printJSON(map[string]any{"symbol": args[0], "callers": callers})
		}
		return nil
	},
}

// ── deps ──────────────────────────────────────────────────────────────────────

var depsCmd = &cobra.Command{
	Use:   "deps <symbol>",
	Short: "Show direct dependencies of a symbol",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		callsFrom, err := graph.ReadFrom(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		deps := callsFrom[args[0]]
		if deps == nil {
			deps = []string{}
		}
		if human {
			fmt.Printf("deps of %s:\n", args[0])
			for _, d := range deps {
				fmt.Printf("  → %s\n", d)
			}
		} else {
			printJSON(map[string]any{"symbol": args[0], "deps": deps})
		}
		return nil
	},
}

// ── show ──────────────────────────────────────────────────────────────────────

var showCmd = &cobra.Command{
	Use:   "show <symbol>",
	Short: "Show source code for a symbol",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		sym, err := findSymbol(repoRoot, args[0])
		if err != nil {
			return err
		}
		absPath := filepath.Join(repoRoot, sym.FilePath)
		block, err := code.Fetch(absPath, sym.StartLine, sym.EndLine)
		if err != nil {
			return err
		}
		block.File = sym.FilePath
		if human {
			fmt.Printf("// %s (%s:%d-%d)\n", sym.Name, sym.FilePath, sym.StartLine, sym.EndLine)
			fmt.Println(block.Code)
		} else {
			printJSON(block)
		}
		return nil
	},
}

// ── search ────────────────────────────────────────────────────────────────────

var searchCmd = &cobra.Command{
	Use:   "search <text>",
	Short: "Search symbols by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		allSyms, err := index.ReadSymbols(repoRoot)
		if err != nil {
			return fmt.Errorf("run 'beakon index' first: %w", err)
		}
		query := strings.ToLower(args[0])
		var matches []pkg.Node
		for _, s := range allSyms {
			if strings.Contains(strings.ToLower(s.Name), query) {
				matches = append(matches, s)
			}
		}
		if human {
			fmt.Printf("search: %s (%d results)\n", args[0], len(matches))
			for _, m := range matches {
				fmt.Printf("  %s  %s:%d\n", m.Name, m.FilePath, m.StartLine)
			}
		} else {
			printJSON(map[string]any{"query": args[0], "results": matches})
		}
		return nil
	},
}

// ── context ───────────────────────────────────────────────────────────────────

var contextCmd = &cobra.Command{
	Use:   "context <symbol>",
	Short: "Assemble complete LLM context for a symbol — anchor + callers + callees + code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoRoot, _ := os.Getwd()
		eng := ctx.NewEngine(repoRoot)
		bundle, err := eng.Assemble(args[0])
		if err != nil {
			return err
		}
		if human {
			printContext(bundle)
		} else {
			printJSON(bundle)
		}
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

// loadSymIndex builds a name→node map for fast lookup during trace enrichment.
func loadSymIndex(repoRoot string) (map[string]pkg.Node, error) {
	syms, err := index.ReadSymbols(repoRoot)
	if err != nil {
		return nil, err
	}
	m := make(map[string]pkg.Node, len(syms))
	for _, s := range syms {
		m[s.Name] = s
	}
	return m, nil
}

// printRichTrace prints trace steps with file location and code snippet.
func printRichTrace(steps []pkg.TraceStep) {
	for _, step := range steps {
		indent := strings.Repeat("  ", step.Depth)
		if step.Depth == 0 {
			fmt.Printf("%s\n", step.Symbol)
		} else {
			fmt.Printf("%s→ %s\n", indent, step.Symbol)
		}
		if step.File != "" {
			fmt.Printf("%s    %s:%d\n", indent, step.File, step.Line)
		}
		if step.Snippet != "" {
			for _, line := range strings.Split(strings.TrimRight(step.Snippet, "\n"), "\n") {
				fmt.Printf("%s    %s\n", indent, line)
			}
			fmt.Println()
		}
	}
}

func findSymbol(repoRoot, name string) (*pkg.Node, error) {
	allSyms, err := index.ReadSymbols(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("run 'beakon index' first: %w", err)
	}
	nameLower := strings.ToLower(name)
	for _, s := range allSyms {
		if strings.ToLower(s.Name) == nameLower {
			return &s, nil
		}
		if strings.HasSuffix(strings.ToLower(s.Name), "."+nameLower) {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("symbol not found: %s", name)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// printContext renders a context bundle in human-readable form.
func printContext(b *ctx.Bundle) {
	sep := strings.Repeat("─", 60)

	fmt.Printf("context: %s\n", b.Query)
	fmt.Printf("files:   %s\n", strings.Join(b.Files, ", "))
	fmt.Printf("tokens:  ~%d\n\n", b.TokenEstimate)

	// Anchor
	fmt.Println(sep)
	printBlock("ANCHOR", b.Anchor)

	// Callees
	if len(b.Callees) > 0 {
		fmt.Println(sep)
		fmt.Printf("CALLS (%d)\n\n", len(b.Callees))
		for _, c := range b.Callees {
			printBlock("→", c)
		}
	}

	// Callers
	if len(b.Callers) > 0 {
		fmt.Println(sep)
		fmt.Printf("CALLED BY (%d)\n\n", len(b.Callers))
		for _, c := range b.Callers {
			printBlock("←", c)
		}
	}

	fmt.Println(sep)
}

// stdlibPkgs are well-known Go stdlib packages — external but not drillable.
var stdlibPkgs = map[string]bool{
	"fmt": true, "os": true, "io": true, "strings": true, "strconv": true,
	"sync": true, "time": true, "path": true, "filepath": true, "errors": true,
	"bytes": true, "bufio": true, "log": true, "math": true, "sort": true,
	"context": true, "net": true, "http": true, "json": true, "encoding": true,
}

func printBlock(label string, b ctx.CodeBlock) {
	if b.Kind == "external" {
		// For pkg.Symbol style names, extract the bare symbol and suggest a drill-in
		// command — unless it's a well-known stdlib package.
		if dot := strings.LastIndex(b.Symbol, "."); dot != -1 {
			pkg := b.Symbol[:dot]
			sym := b.Symbol[dot+1:]
			if !stdlibPkgs[pkg] {
				fmt.Printf("%s  %s  (external — ./beakon context %s)\n\n", label, b.Symbol, sym)
				return
			}
		}
		fmt.Printf("%s  %s  (external)\n\n", label, b.Symbol)
		return
	}
	fmt.Printf("%s  %s\n", label, b.Symbol)
	fmt.Printf("     %s:%d-%d\n\n", b.File, b.Start, b.End)
	if b.Code != "" {
		for _, line := range strings.Split(b.Code, "\n") {
			fmt.Printf("     %s\n", line)
		}
		fmt.Println()
	}
}

func init() {
	root.PersistentFlags().BoolVar(&human, "human", false, "Human-readable output")
	root.AddCommand(indexCmd, watchCmd, mapCmd, traceCmd, explainCmd,
		callersCmd, depsCmd, showCmd, searchCmd, contextCmd)
}

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
