package symbols

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/codeindex/codeindex/pkg"
)

// Extract parses a source file and returns all symbols and call edges.
func Extract(filePath, language string, source []byte) ([]pkg.CodeIndexNode, []pkg.CallEdge) {
	switch language {
	case "go":
		return extractGo(filePath, source)
	case "typescript", "javascript":
		return extractTS(filePath, language, source)
	case "python":
		return extractPython(filePath, source)
	}
	return nil, nil
}

// HashFile returns the sha1 hash of a file's contents.
func HashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha1.Sum(data)
	return hex.EncodeToString(h[:])
}

// --- Go ---

func extractGo(filePath string, src []byte) ([]pkg.CodeIndexNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "go")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.CodeIndexNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_declaration":
			name, node := goFuncName(n, src)
			if name != "" {
				nodes = append(nodes, pkg.CodeIndexNode{
					ID:         pkg.NodeID("go", "function", filePath, name),
					Kind:       "function",
					Name:       name,
					Language:   "go",
					FilePath:   filePath,
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				calls = append(calls, goCallEdges(n, src, name)...)
			}
		case "method_declaration":
			name, receiver, node := goMethodName(n, src)
			if name != "" {
				qualified := name
				if receiver != "" {
					qualified = receiver + "." + name
				}
				nodes = append(nodes, pkg.CodeIndexNode{
					ID:         pkg.NodeID("go", "method", filePath, qualified),
					Kind:       "method",
					Name:       qualified,
					Language:   "go",
					FilePath:   filePath,
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					Parent:     receiver,
					SourceHash: hash,
				})
				calls = append(calls, goCallEdges(n, src, qualified)...)
			}
		case "type_declaration":
			name, node := goTypeName(n, src)
			if name != "" {
				nodes = append(nodes, pkg.CodeIndexNode{
					ID:         pkg.NodeID("go", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "go",
					FilePath:   filePath,
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					SourceHash: hash,
				})
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}

	walk(tree.RootNode(), "")
	return nodes, calls
}

func goFuncName(n *sitter.Node, src []byte) (string, *sitter.Node) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return "", nil
	}
	return nameNode.Content(src), n
}

func goMethodName(n *sitter.Node, src []byte) (string, string, *sitter.Node) {
	nameNode := n.ChildByFieldName("name")
	recvNode := n.ChildByFieldName("receiver")
	if nameNode == nil {
		return "", "", nil
	}
	receiver := ""
	if recvNode != nil {
		receiver = extractGoReceiver(recvNode.Content(src))
	}
	return nameNode.Content(src), receiver, n
}

func goTypeName(n *sitter.Node, src []byte) (string, *sitter.Node) {
	for i := 0; i < int(n.ChildCount()); i++ {
		spec := n.Child(i)
		if spec.Type() == "type_spec" {
			nameNode := spec.ChildByFieldName("name")
			if nameNode != nil {
				return nameNode.Content(src), n
			}
		}
	}
	return "", nil
}

func goCallEdges(n *sitter.Node, src []byte, from string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}

func extractGoReceiver(recv string) string {
	recv = strings.Trim(recv, "()")
	for _, part := range strings.Fields(recv) {
		part = strings.TrimPrefix(part, "*")
		if len(part) > 0 && part[0] >= 'A' && part[0] <= 'Z' {
			return part
		}
	}
	return ""
}

// --- TypeScript / JavaScript ---

func extractTS(filePath, language string, src []byte) ([]pkg.CodeIndexNode, []pkg.CallEdge) {
	tree, err := parseSource(src, language)
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.CodeIndexNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.CodeIndexNode{
					ID: pkg.NodeID(language, "function", filePath, name),
					Kind: "function", Name: name, Language: language,
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				calls = append(calls, tsCallEdges(n, src, name)...)
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "class_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.CodeIndexNode{
					ID: pkg.NodeID(language, "class", filePath, name),
					Kind: "class", Name: name, Language: language,
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "method_definition":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				nodes = append(nodes, pkg.CodeIndexNode{
					ID: pkg.NodeID(language, "method", filePath, qualified),
					Kind: "method", Name: qualified, Language: language,
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					Parent:    parent,
					SourceHash: hash,
				})
				calls = append(calls, tsCallEdges(n, src, qualified)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}

	walk(tree.RootNode(), "")
	return nodes, calls
}

func tsCallEdges(n *sitter.Node, src []byte, from string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}

// --- Python ---

func extractPython(filePath string, src []byte) ([]pkg.CodeIndexNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "python")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.CodeIndexNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_definition":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				kind := "function"
				qualified := name
				if parent != "" {
					kind = "method"
					qualified = parent + "." + name
				}
				nodes = append(nodes, pkg.CodeIndexNode{
					ID: pkg.NodeID("python", kind, filePath, qualified),
					Kind: kind, Name: qualified, Language: "python",
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					Parent:    parent,
					SourceHash: hash,
				})
				calls = append(calls, pyCallEdges(n, src, qualified)...)
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), qualified)
				}
				return
			}
		case "class_definition":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.CodeIndexNode{
					ID: pkg.NodeID("python", "class", filePath, name),
					Kind: "class", Name: name, Language: "python",
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}

	walk(tree.RootNode(), "")
	return nodes, calls
}

func pyCallEdges(n *sitter.Node, src []byte, from string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}

// --- Helpers ---

func hashBytes(data []byte) string {
	h := sha1.Sum(data)
	return hex.EncodeToString(h[:])
}

func parseSource(src []byte, language string) (*sitter.Tree, error) {
	// Imported from parser package
	return getParser(language, src)
}
