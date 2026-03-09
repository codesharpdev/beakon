#!/bin/bash
# CodeIndex — Local Setup Script
# Run this once on your machine to bootstrap the project.
# Requires: Go 1.21+

set -e

echo "==> Creating project structure..."
mkdir -p codeindex
cd codeindex

echo "==> Initializing Go module..."
go mod init github.com/codeindex/codeindex

echo "==> Creating directory structure..."
mkdir -p \
  cmd/codeindex \
  internal/types \
  internal/discover \
  internal/parser \
  internal/indexer \
  internal/store \
  internal/query \
  internal/toon \
  internal/watch \
  internal/graph \
  testdata/sample_repo/auth \
  testdata/sample_repo/api \
  docs

echo "==> Installing dependencies..."
go get github.com/spf13/cobra@v1.8.0
go get github.com/smacker/go-tree-sitter@v0.0.0-20231219031718-233c2f923ac7
go get github.com/fsnotify/fsnotify@v1.7.0
go get gopkg.in/yaml.v3@v3.0.1

echo "==> Adding .gitignore..."
cat > .gitignore << 'EOF'
.codeindex/nodes/
.codeindex/deps/
.codeindex/meta.json
.codeindex/toc.json
bin/
*.test
EOF

echo "==> Adding .codeindex config placeholder..."
mkdir -p .codeindex
cat > .codeindex/config.yaml << 'EOF'
version: 1
languages:
  - go
  - typescript
  - javascript
  - python
ignore:
  - vendor/
  - node_modules/
  - dist/
  - build/
EOF

echo ""
echo "✓ Setup complete."
echo ""
echo "Next steps:"
echo "  1. Copy all source files into the project (or let Claude Code generate them)"
echo "  2. go mod tidy"
echo "  3. go build ./cmd/codeindex"
echo "  4. ./codeindex init"
echo "  5. ./codeindex index"
echo "  6. ./codeindex map"
