package code

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestFetch_BasicRange(t *testing.T) {
	path := writeTemp(t, "line1\nline2\nline3\nline4\n")
	block, err := Fetch(path, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if block.Code != "line2\nline3" {
		t.Errorf("got %q", block.Code)
	}
	if block.Start != 2 || block.End != 3 {
		t.Errorf("range = %d-%d, want 2-3", block.Start, block.End)
	}
	if block.File != path {
		t.Errorf("file = %q, want %q", block.File, path)
	}
}

func TestFetch_SingleLine(t *testing.T) {
	path := writeTemp(t, "only\n")
	block, err := Fetch(path, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if block.Code != "only" {
		t.Errorf("got %q", block.Code)
	}
}

func TestFetch_ClampsBelowStart(t *testing.T) {
	path := writeTemp(t, "a\nb\nc\n")
	block, err := Fetch(path, -5, 2)
	if err != nil {
		t.Fatal(err)
	}
	// start clamped to 1
	if block.Start != 1 {
		t.Errorf("start = %d, want 1", block.Start)
	}
}

func TestFetch_ClampsAboveEnd(t *testing.T) {
	path := writeTemp(t, "a\nb\nc\n")
	block, err := Fetch(path, 1, 999)
	if err != nil {
		t.Fatal(err)
	}
	if block.End > 4 { // "a\nb\nc\n" splits into 4 elements (trailing empty)
		t.Errorf("end not clamped: %d", block.End)
	}
}

func TestFetch_InvalidRange(t *testing.T) {
	path := writeTemp(t, "a\nb\n")
	_, err := Fetch(path, 5, 10)
	if err == nil {
		t.Error("expected error for out-of-bounds range")
	}
}

func TestFetch_StartAfterEnd(t *testing.T) {
	path := writeTemp(t, "a\nb\nc\n")
	_, err := Fetch(path, 3, 1)
	if err == nil {
		t.Error("expected error when start > end")
	}
}

func TestFetch_MissingFile(t *testing.T) {
	_, err := Fetch(filepath.Join(t.TempDir(), "nonexistent.go"), 1, 1)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
