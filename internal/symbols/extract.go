package symbols

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/codesharpdev/beakon/pkg"
)

// Extract parses a source file and returns all symbols and call edges.
func Extract(filePath, language string, source []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	switch language {
	case "go":
		return extractGo(filePath, source)
	case "typescript", "javascript":
		return extractTS(filePath, language, source)
	case "python":
		return extractPython(filePath, source)
	case "rust":
		return extractRust(filePath, source)
	case "java", "groovy":
		return extractJava(filePath, language, source)
	case "c", "cpp":
		return extractC(filePath, language, source)
	case "csharp":
		return extractCSharp(filePath, source)
	case "ruby":
		return extractRuby(filePath, source)
	case "kotlin":
		return extractKotlin(filePath, source)
	case "swift":
		return extractSwift(filePath, source)
	case "php":
		return extractPHP(filePath, source)
	case "scala":
		return extractScala(filePath, source)
	case "elixir":
		return extractElixir(filePath, source)
	case "ocaml":
		return extractOCaml(filePath, source)
	case "elm":
		return extractElm(filePath, source)
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

func extractGo(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "go")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_declaration":
			name, node := goFuncName(n, src)
			if name != "" {
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("go", "function", filePath, name),
					Kind:       "function",
					Name:       name,
					Language:   "go",
					FilePath:   filePath,
					StartLine:  int(node.StartPoint().Row) + 1,
					EndLine:    int(node.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				calls = append(calls, goCallEdges(n, src, name, "")...)
			}
		case "method_declaration":
			name, receiver, node := goMethodName(n, src)
			if name != "" {
				qualified := name
				if receiver != "" {
					qualified = receiver + "." + name
				}
				nodes = append(nodes, pkg.BeakonNode{
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
				calls = append(calls, goCallEdges(n, src, qualified, receiver)...)
			}
		case "type_declaration":
			name, node := goTypeName(n, src)
			if name != "" {
				nodes = append(nodes, pkg.BeakonNode{
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

func goCallEdges(n *sitter.Node, src []byte, from string, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// qualifyCall rewrites "s.Foo" or "self.Foo" to "ReceiverType.Foo" when receiverType is known.
func qualifyCall(callee, receiverType string) string {
	if receiverType == "" {
		return callee
	}
	dot := strings.Index(callee, ".")
	if dot <= 0 {
		return callee
	}
	prefix := callee[:dot]
	method := callee[dot+1:]
	// Short lowercase prefix → likely a receiver variable (s, r, self, this)
	if len(prefix) <= 6 && len(prefix) > 0 && prefix[0] >= 'a' && prefix[0] <= 'z' {
		return receiverType + "." + method
	}
	return callee
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

func extractTS(filePath, language string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, language)
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID: pkg.NodeID(language, "function", filePath, name),
					Kind: "function", Name: name, Language: language,
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				calls = append(calls, tsCallEdges(n, src, name, "")...)
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "class_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
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
				nodes = append(nodes, pkg.BeakonNode{
					ID: pkg.NodeID(language, "method", filePath, qualified),
					Kind: "method", Name: qualified, Language: language,
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					Parent:    parent,
					SourceHash: hash,
				})
				calls = append(calls, tsCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}

	walk(tree.RootNode(), "")
	return nodes, calls
}

func tsCallEdges(n *sitter.Node, src []byte, from string, parent string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					if parent != "" && strings.HasPrefix(callee, "this.") {
						callee = parent + "." + callee[5:]
					}
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

func extractPython(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "python")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
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
				nodes = append(nodes, pkg.BeakonNode{
					ID: pkg.NodeID("python", kind, filePath, qualified),
					Kind: kind, Name: qualified, Language: "python",
					FilePath: filePath,
					StartLine: int(n.StartPoint().Row) + 1,
					EndLine:   int(n.EndPoint().Row) + 1,
					Parent:    parent,
					SourceHash: hash,
				})
				calls = append(calls, pyCallEdges(n, src, qualified, parent)...)
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), qualified)
				}
				return
			}
		case "class_definition":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
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

func pyCallEdges(n *sitter.Node, src []byte, from string, parent string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					if parent != "" && strings.HasPrefix(callee, "self.") {
						callee = parent + "." + callee[5:]
					}
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

// --- Rust ---

func extractRust(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "rust")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_item":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("rust", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "rust",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, rustCallEdges(n, src, qualified, parent)...)
			}
		case "struct_item", "enum_item", "trait_item":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("rust", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "rust",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "impl_item":
			typeNode := n.ChildByFieldName("type")
			implParent := ""
			if typeNode != nil {
				implParent = strings.TrimSpace(typeNode.Content(src))
			}
			for i := 0; i < int(n.ChildCount()); i++ {
				walk(n.Child(i), implParent)
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func rustCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- Java / Groovy ---

func extractJava(filePath, language string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, language)
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID(language, "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   language,
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "method_declaration", "constructor_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "method"
				if parent == "" {
					kind = "function"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID(language, kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   language,
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, javaCallEdges(n, src, qualified, parent, language)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func javaCallEdges(n *sitter.Node, src []byte, from, receiver, language string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			nameNode := node.ChildByFieldName("name")
			objNode := node.ChildByFieldName("object")
			if nameNode != nil {
				callee := nameNode.Content(src)
				if objNode != nil {
					obj := strings.TrimSpace(objNode.Content(src))
					if obj == "this" && receiver != "" {
						callee = receiver + "." + callee
					} else {
						callee = obj + "." + callee
					}
				} else if receiver != "" {
					callee = receiver + "." + callee
				}
				if callee != from {
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

// --- C / C++ ---

func extractC(filePath, language string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, language)
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "function_definition":
			name := cFuncName(n, src)
			if name != "" {
				qualified := name
				if parent != "" {
					qualified = parent + "::" + name
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID(language, "function", filePath, qualified),
					Kind:       "function",
					Name:       qualified,
					Language:   language,
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, cCallEdges(n, src, qualified, parent)...)
			}
		case "struct_specifier", "class_specifier":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID(language, "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   language,
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
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

// cFuncName extracts the function name from a C/C++ function_definition node.
// The name is buried in nested declarator nodes.
func cFuncName(n *sitter.Node, src []byte) string {
	decl := n.ChildByFieldName("declarator")
	for decl != nil {
		if decl.Type() == "function_declarator" {
			inner := decl.ChildByFieldName("declarator")
			if inner != nil {
				return strings.TrimSpace(inner.Content(src))
			}
		}
		// Recurse into pointer declarators etc.
		next := decl.ChildByFieldName("declarator")
		if next == decl {
			break
		}
		decl = next
	}
	return ""
}

func cCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
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

// --- C# ---

func extractCSharp(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "csharp")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_declaration", "interface_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("csharp", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "csharp",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "method_declaration", "constructor_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "method"
				if parent == "" {
					kind = "function"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("csharp", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "csharp",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, csharpCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func csharpCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "invocation_expression" {
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- Ruby ---

func extractRuby(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "ruby")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class", "module":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("ruby", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "ruby",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "method", "singleton_method":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("ruby", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "ruby",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, rubyCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func rubyCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call" {
			methodNode := node.ChildByFieldName("method")
			if methodNode != nil {
				callee := strings.TrimSpace(methodNode.Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- Kotlin ---

func extractKotlin(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "kotlin")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	// Helper: find first simple_identifier child
	firstIdent := func(n *sitter.Node) string {
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c.Type() == "simple_identifier" {
				return c.Content(src)
			}
		}
		return ""
	}

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_declaration":
			name := firstIdent(n)
			if name != "" {
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("kotlin", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "kotlin",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "function_declaration":
			name := firstIdent(n)
			if name != "" {
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("kotlin", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "kotlin",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, kotlinCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func kotlinCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			callee := strings.TrimSpace(node.Child(0).Content(src))
			if callee != "" && callee != from {
				callee = qualifyCall(callee, receiver)
				edges = append(edges, pkg.CallEdge{From: from, To: callee})
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(n)
	return edges
}

// --- Swift ---

func extractSwift(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "swift")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	// Helper: get name from simple_identifier or type_identifier child
	firstName := func(n *sitter.Node) string {
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c.Type() == "simple_identifier" || c.Type() == "type_identifier" {
				return c.Content(src)
			}
		}
		return ""
	}

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_declaration", "struct_declaration":
			name := firstName(n)
			if name != "" {
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("swift", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "swift",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "function_declaration":
			name := firstName(n)
			if name != "" {
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("swift", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "swift",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, swiftCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func swiftCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			if node.ChildCount() > 0 {
				callee := strings.TrimSpace(node.Child(0).Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- PHP ---

func extractPHP(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "php")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("php", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "php",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "function_definition", "method_declaration":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("php", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "php",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, phpCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func phpCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		switch node.Type() {
		case "function_call_expression":
			fn := node.ChildByFieldName("function")
			if fn != nil {
				callee := strings.TrimSpace(fn.Content(src))
				if callee != "" && callee != from {
					edges = append(edges, pkg.CallEdge{From: from, To: callee})
				}
			}
		case "member_call_expression":
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				callee := strings.TrimSpace(nameNode.Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- Scala ---

func extractScala(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "scala")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	// Helper: find first identifier child
	firstIdent := func(n *sitter.Node) string {
		for i := 0; i < int(n.ChildCount()); i++ {
			c := n.Child(i)
			if c.Type() == "identifier" {
				return c.Content(src)
			}
		}
		return ""
	}

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "class_definition", "object_definition", "trait_definition":
			name := firstIdent(n)
			if name != "" {
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("scala", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "scala",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "function_definition":
			name := firstIdent(n)
			if name != "" {
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("scala", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "scala",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, scalaCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func scalaCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			if node.ChildCount() > 0 {
				callee := strings.TrimSpace(node.Child(0).Content(src))
				if callee != "" && callee != from {
					callee = qualifyCall(callee, receiver)
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

// --- Elixir ---

func extractElixir(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "elixir")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	// Elixir is macro-based: def/defp/defmodule are call nodes.
	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		if n.Type() == "call" {
			// The target (function name of the macro call) is the first child
			if n.ChildCount() > 0 {
				target := n.Child(0)
				targetName := strings.TrimSpace(target.Content(src))
				switch targetName {
				case "defmodule":
					// args is second child; first arg is the module name
					if n.ChildCount() > 1 {
						argsNode := n.Child(1)
						moduleName := ""
						if argsNode.ChildCount() > 0 {
							moduleName = strings.TrimSpace(argsNode.Child(0).Content(src))
						} else {
							moduleName = strings.TrimSpace(argsNode.Content(src))
						}
						if moduleName != "" {
							nodes = append(nodes, pkg.BeakonNode{
								ID:         pkg.NodeID("elixir", "class", filePath, moduleName),
								Kind:       "class",
								Name:       moduleName,
								Language:   "elixir",
								FilePath:   filePath,
								StartLine:  int(n.StartPoint().Row) + 1,
								EndLine:    int(n.EndPoint().Row) + 1,
								SourceHash: hash,
							})
							for i := 0; i < int(n.ChildCount()); i++ {
								walk(n.Child(i), moduleName)
							}
							return
						}
					}
				case "def", "defp":
					if n.ChildCount() > 1 {
						argsNode := n.Child(1)
						funcName := ""
						if argsNode.ChildCount() > 0 {
							funcName = strings.TrimSpace(argsNode.Child(0).Content(src))
						} else {
							funcName = strings.TrimSpace(argsNode.Content(src))
						}
						if funcName != "" {
							qualified := funcName
							if parent != "" {
								qualified = parent + "." + funcName
							}
							kind := "function"
							if parent != "" {
								kind = "method"
							}
							nodes = append(nodes, pkg.BeakonNode{
								ID:         pkg.NodeID("elixir", kind, filePath, qualified),
								Kind:       kind,
								Name:       qualified,
								Language:   "elixir",
								FilePath:   filePath,
								StartLine:  int(n.StartPoint().Row) + 1,
								EndLine:    int(n.EndPoint().Row) + 1,
								Parent:     parent,
								SourceHash: hash,
							})
						}
					}
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

// --- OCaml ---

func extractOCaml(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "ocaml")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "module_definition":
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("ocaml", "class", filePath, name),
					Kind:       "class",
					Name:       name,
					Language:   "ocaml",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					SourceHash: hash,
				})
				for i := 0; i < int(n.ChildCount()); i++ {
					walk(n.Child(i), name)
				}
				return
			}
		case "let_binding":
			// A let_binding is a function if it has parameter children
			nameNode := n.ChildByFieldName("pattern")
			hasParams := false
			for i := 0; i < int(n.ChildCount()); i++ {
				if n.Child(i).Type() == "parameter" {
					hasParams = true
					break
				}
			}
			if nameNode != nil && hasParams {
				name := strings.TrimSpace(nameNode.Content(src))
				qualified := name
				if parent != "" {
					qualified = parent + "." + name
				}
				kind := "function"
				if parent != "" {
					kind = "method"
				}
				nodes = append(nodes, pkg.BeakonNode{
					ID:         pkg.NodeID("ocaml", kind, filePath, qualified),
					Kind:       kind,
					Name:       qualified,
					Language:   "ocaml",
					FilePath:   filePath,
					StartLine:  int(n.StartPoint().Row) + 1,
					EndLine:    int(n.EndPoint().Row) + 1,
					Parent:     parent,
					SourceHash: hash,
				})
				calls = append(calls, ocamlCallEdges(n, src, qualified, parent)...)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func ocamlCallEdges(n *sitter.Node, src []byte, from, receiver string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "application" {
			if node.ChildCount() > 0 {
				callee := strings.TrimSpace(node.Child(0).Content(src))
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

// --- Elm ---

func extractElm(filePath string, src []byte) ([]pkg.BeakonNode, []pkg.CallEdge) {
	tree, err := parseSource(src, "elm")
	if err != nil {
		return nil, nil
	}
	hash := hashBytes(src)
	var nodes []pkg.BeakonNode
	var calls []pkg.CallEdge

	var walk func(n *sitter.Node, parent string)
	walk = func(n *sitter.Node, parent string) {
		switch n.Type() {
		case "type_declaration":
			// Find the type name
			for i := 0; i < int(n.ChildCount()); i++ {
				c := n.Child(i)
				if c.Type() == "upper_case_identifier" {
					name := c.Content(src)
					nodes = append(nodes, pkg.BeakonNode{
						ID:         pkg.NodeID("elm", "class", filePath, name),
						Kind:       "class",
						Name:       name,
						Language:   "elm",
						FilePath:   filePath,
						StartLine:  int(n.StartPoint().Row) + 1,
						EndLine:    int(n.EndPoint().Row) + 1,
						SourceHash: hash,
					})
					break
				}
			}
		case "value_declaration":
			// Look for function_declaration_left child to get function name
			for i := 0; i < int(n.ChildCount()); i++ {
				c := n.Child(i)
				if c.Type() == "function_declaration_left" {
					nameNode := c.Child(0)
					if nameNode != nil {
						name := strings.TrimSpace(nameNode.Content(src))
						if name != "" {
							nodes = append(nodes, pkg.BeakonNode{
								ID:         pkg.NodeID("elm", "function", filePath, name),
								Kind:       "function",
								Name:       name,
								Language:   "elm",
								FilePath:   filePath,
								StartLine:  int(n.StartPoint().Row) + 1,
								EndLine:    int(n.EndPoint().Row) + 1,
								SourceHash: hash,
							})
							calls = append(calls, elmCallEdges(n, src, name)...)
						}
					}
					break
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), parent)
		}
	}
	walk(tree.RootNode(), "")
	return nodes, calls
}

func elmCallEdges(n *sitter.Node, src []byte, from string) []pkg.CallEdge {
	var edges []pkg.CallEdge
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "function_call_expr" {
			if node.ChildCount() > 0 {
				callee := strings.TrimSpace(node.Child(0).Content(src))
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
