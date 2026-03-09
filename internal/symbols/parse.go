package symbols

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
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
