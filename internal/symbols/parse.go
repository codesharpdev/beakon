package symbols

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func treeSitterLang(language string) (*sitter.Language, error) {
	switch language {
	case "go":
		return golang.GetLanguage(), nil
	case "typescript":
		return typescript.GetLanguage(), nil
	case "javascript":
		return javascript.GetLanguage(), nil
	case "python":
		return python.GetLanguage(), nil
	case "rust":
		return rust.GetLanguage(), nil
	case "java":
		return java.GetLanguage(), nil
	case "c":
		return c.GetLanguage(), nil
	case "cpp":
		return cpp.GetLanguage(), nil
	case "csharp":
		return csharp.GetLanguage(), nil
	case "ruby":
		return ruby.GetLanguage(), nil
	case "kotlin":
		return kotlin.GetLanguage(), nil
	case "swift":
		return swift.GetLanguage(), nil
	case "php":
		return php.GetLanguage(), nil
	case "scala":
		return scala.GetLanguage(), nil
	case "elixir":
		return elixir.GetLanguage(), nil
	case "ocaml":
		return ocaml.GetLanguage(), nil
	case "elm":
		return elm.GetLanguage(), nil
	case "groovy":
		return groovy.GetLanguage(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// getParser creates a new parser per call so concurrent goroutines don't share state.
// Tree-sitter parsers are not thread-safe; a global parser causes C assertion failures.
func getParser(language string, src []byte) (*sitter.Tree, error) { //nolint:unused
	lang, err := treeSitterLang(language)
	if err != nil {
		return nil, err
	}
	p := sitter.NewParser()
	p.SetLanguage(lang)
	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return tree, nil
}
