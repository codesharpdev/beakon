package symbols

import (
	"testing"

	"github.com/codesharpdev/beakon/pkg"
)

const goSrc = `package auth

type AuthService struct{}

func Login(user, pass string) bool {
	return validatePassword(user, pass)
}

func (s *AuthService) Logout(user string) {
	clearSession(user)
}
`

const tsSrc = `
class AuthService {
  login(user) {
    return this.validatePassword(user);
  }
}

function logout(user) {
  clearSession(user);
}
`

const pySrc = `
class AuthService:
    def login(self, user, password):
        return self.validate(user, password)

def logout(user):
    clear_session(user)
`

func TestExtractGo_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.go", "go", []byte(goSrc))
	names := symbolNames(nodes)
	assertContains(t, names, "Login")
	assertContains(t, names, "AuthService.Logout")
	assertContains(t, names, "AuthService")
}

func TestExtractGo_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.go", "go", []byte(goSrc))
	assertEdge(t, calls, "Login", "validatePassword")
	assertEdge(t, calls, "AuthService.Logout", "clearSession")
}

func TestExtractGo_LineNumbers(t *testing.T) {
	nodes, _ := Extract("auth/service.go", "go", []byte(goSrc))
	for _, n := range nodes {
		if n.Name == "Login" {
			if n.StartLine < 1 {
				t.Errorf("Login StartLine = %d, want >= 1", n.StartLine)
			}
			if n.EndLine < n.StartLine {
				t.Errorf("Login EndLine %d < StartLine %d", n.EndLine, n.StartLine)
			}
		}
	}
}

func TestExtractTS_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.ts", "typescript", []byte(tsSrc))
	names := symbolNames(nodes)
	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_Symbols(t *testing.T) {
	nodes, _ := Extract("auth/service.py", "python", []byte(pySrc))
	names := symbolNames(nodes)
	assertContains(t, names, "AuthService")
	assertContains(t, names, "AuthService.login")
	assertContains(t, names, "logout")
}

func TestExtractPython_CallEdges(t *testing.T) {
	_, calls := Extract("auth/service.py", "python", []byte(pySrc))
	assertEdge(t, calls, "logout", "clear_session")
}

func TestExtract_UnknownLanguage(t *testing.T) {
	nodes, calls := Extract("file.lua", "lua", []byte("function foo() end"))
	if len(nodes) != 0 || len(calls) != 0 {
		t.Error("expected empty results for unknown language")
	}
}

// --- call edge qualification tests ---

const goMethodCallSrc = `package auth

type AuthService struct{}

func (s *AuthService) Login(user string) bool {
	return s.validatePassword(user)
}

func (s *AuthService) validatePassword(user string) bool {
	return true
}
`

func TestExtractGo_MethodCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.go", "go", []byte(goMethodCallSrc))
	assertEdge(t, calls, "AuthService.Login", "AuthService.validatePassword")
}

const tsMethodCallSrc = `
class AuthService {
  login(user) {
    return this.validatePassword(user);
  }
  validatePassword(user) { return true; }
}
`

func TestExtractTS_ThisCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.ts", "typescript", []byte(tsMethodCallSrc))
	assertEdge(t, calls, "AuthService.login", "AuthService.validatePassword")
}

const pyMethodCallSrc = `
class AuthService:
    def login(self, user):
        return self.validate(user)
    def validate(self, user):
        return True
`

func TestExtractPython_SelfCallQualified(t *testing.T) {
	_, calls := Extract("auth/service.py", "python", []byte(pyMethodCallSrc))
	assertEdge(t, calls, "AuthService.login", "AuthService.validate")
}

// --- helpers ---

func symbolNames(nodes []pkg.BeakonNode) []string {
	var names []string
	for _, n := range nodes {
		names = append(names, n.Name)
	}
	return names
}

func assertContains(t *testing.T, names []string, want string) {
	t.Helper()
	for _, n := range names {
		if n == want {
			return
		}
	}
	t.Errorf("symbol %q not found in %v", want, names)
}

func assertEdge(t *testing.T, calls []pkg.CallEdge, from, to string) {
	t.Helper()
	for _, c := range calls {
		if c.From == from && c.To == to {
			return
		}
	}
	t.Errorf("call edge %q → %q not found", from, to)
}
