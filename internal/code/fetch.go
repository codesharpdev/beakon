package code

import (
	"fmt"
	"os"
	"strings"
)

// Block holds a code block extracted from a source file.
type Block struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Code  string `json:"code"`
}

// Fetch reads lines [start, end] from file (1-indexed, inclusive).
func Fetch(filePath string, start, end int) (*Block, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	lines := strings.Split(string(data), "\n")

	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end || start > len(lines) {
		return nil, fmt.Errorf("invalid line range %d-%d for %d lines", start, end, len(lines))
	}

	code := strings.Join(lines[start-1:end], "\n")

	return &Block{
		File:  filePath,
		Start: start,
		End:   end,
		Code:  code,
	}, nil
}
