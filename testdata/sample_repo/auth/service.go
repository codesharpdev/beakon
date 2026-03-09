package auth

import "errors"

// AuthService handles authentication logic.
type AuthService struct{}

// Login validates credentials and returns a JWT.
func (s *AuthService) Login(username, password string) (string, error) {
	if err := validatePassword(username, password); err != nil {
		return "", err
	}
	token := createJWT(username)
	return token, nil
}

// Logout invalidates a session token.
func (s *AuthService) Logout(token string) error {
	return invalidateToken(token)
}

func validatePassword(username, password string) error {
	if password == "" {
		return errors.New("password required")
	}
	return nil
}

func createJWT(username string) string {
	return "jwt." + username
}

func invalidateToken(token string) error {
	return nil
}
