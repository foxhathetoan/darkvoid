package dto

// LoginRequest represents the data required for authentication.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response after successful authentication.
type LoginResponse struct {
	AccessToken      string       `json:"access_token"`
	RefreshToken     string       `json:"refresh_token,omitempty"`
	TokenType        string       `json:"token_type"`
	AccessExpiresIn  int64        `json:"access_expires_in"`
	RefreshExpiresIn int64        `json:"refresh_expires_in"`
	User             UserResponse `json:"user"`
}

// RefreshTokenRequest represents the request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents the response after refreshing tokens.
type RefreshTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	TokenType        string `json:"token_type"`
	AccessExpiresIn  int64  `json:"access_expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
}

// LogoutRequest represents the request to logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RegisterRequest represents the data required to register a new account.
type RegisterRequest struct {
	Username    string `json:"username" example:"johndoe"`
	Email       string `json:"email" example:"john@example.com"`
	DisplayName string `json:"display_name" example:"John Doe"`
	Password    string `json:"password" example:"SecurePass123"`
}

// RegisterResponse is returned after a successful registration.
// Contains tokens so the user is immediately logged in.
type RegisterResponse struct {
	UserID           string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	TokenType        string `json:"token_type"`
	AccessExpiresIn  int64  `json:"access_expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
}

// ChangePasswordRequest represents the request to change password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
