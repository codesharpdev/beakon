package api

import "github.com/codeindex/codeindex/testdata/sample_repo/auth"

// UserController handles HTTP requests for user operations.
type UserController struct {
	auth *auth.AuthService
}

// Login handles POST /login
func (c *UserController) Login(username, password string) (string, error) {
	return c.auth.Login(username, password)
}

// Logout handles POST /logout
func (c *UserController) Logout(token string) error {
	return c.auth.Logout(token)
}
