package graph

import (
	"path/filepath"
	"testing"

	"github.com/beakon/beakon/pkg"
)

func edges(pairs ...string) []pkg.CallEdge {
	var out []pkg.CallEdge
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, pkg.CallEdge{From: pairs[i], To: pairs[i+1]})
	}
	return out
}

func TestBuild_DirectionsCorrect(t *testing.T) {
	from, to := Build(edges(
		"Login", "validatePassword",
		"Login", "createJWT",
		"Logout", "clearSession",
	))

	assertSliceContains(t, from["Login"], "validatePassword")
	assertSliceContains(t, from["Login"], "createJWT")
	assertSliceContains(t, from["Logout"], "clearSession")

	assertSliceContains(t, to["validatePassword"], "Login")
	assertSliceContains(t, to["createJWT"], "Login")
	assertSliceContains(t, to["clearSession"], "Logout")
}

func TestBuild_Deduplicates(t *testing.T) {
	from, _ := Build(edges(
		"Login", "validatePassword",
		"Login", "validatePassword", // duplicate
	))
	if len(from["Login"]) != 1 {
		t.Errorf("expected 1 unique callee, got %d", len(from["Login"]))
	}
}

func TestTrace_BFS(t *testing.T) {
	from, _ := Build(edges(
		"Login", "validatePassword",
		"validatePassword", "hashPassword",
	))
	result := Trace("Login", from)
	assertSliceContains(t, result, "Login")
	assertSliceContains(t, result, "validatePassword")
	assertSliceContains(t, result, "hashPassword")
}

func TestTrace_CycleDetection(t *testing.T) {
	from, _ := Build(edges(
		"A", "B",
		"B", "A", // cycle
	))
	result := Trace("A", from)
	// Should not loop forever; should have exactly 2 unique symbols
	if len(result) != 2 {
		t.Errorf("expected 2 results with cycle, got %d: %v", len(result), result)
	}
}

func TestWriteRead_RoundTrip(t *testing.T) {
	root := t.TempDir()
	from, to := Build(edges("Login", "validatePassword"))

	if err := Write(root, from, to); err != nil {
		t.Fatal(err)
	}

	gotFrom, err := ReadFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	gotTo, err := ReadTo(root)
	if err != nil {
		t.Fatal(err)
	}

	assertSliceContains(t, gotFrom["Login"], "validatePassword")
	assertSliceContains(t, gotTo["validatePassword"], "Login")
}

func TestWrite_NoTmpFilesLeft(t *testing.T) {
	root := t.TempDir()
	from, to := Build(edges("A", "B"))
	if err := Write(root, from, to); err != nil {
		t.Fatal(err)
	}
	entries, _ := filepath.Glob(filepath.Join(root, graphDir, "*.tmp"))
	if len(entries) > 0 {
		t.Errorf("tmp files left after Write: %v", entries)
	}
}

func TestImpact_ReverseBFS(t *testing.T) {
	_, to := Build(edges(
		"Login", "validatePassword",
		"Login", "createJWT",
		"Handle", "Login",
	))

	result := Impact("validatePassword", to)

	assertSliceContains(t, result, "validatePassword")
	assertSliceContains(t, result, "Login")
	assertSliceContains(t, result, "Handle")
}

func TestImpact_CycleDetection(t *testing.T) {
	_, to := Build(edges(
		"A", "B",
		"B", "A",
	))
	result := Impact("A", to)
	if len(result) > 2 {
		t.Errorf("cycle not detected: got %d results", len(result))
	}
}

func TestImpact_IsolatedSymbol(t *testing.T) {
	_, to := Build(edges("A", "B"))
	result := Impact("A", to) // nothing calls A
	if len(result) != 1 {
		t.Errorf("expected only anchor symbol, got %v", result)
	}
}

func assertSliceContains(t *testing.T, slice []string, want string) {
	t.Helper()
	for _, s := range slice {
		if s == want {
			return
		}
	}
	t.Errorf("%q not found in %v", want, slice)
}
