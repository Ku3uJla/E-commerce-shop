package dto

import "time"

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse is returned after a successful login.
type AuthResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// ErrorResponse is the unified error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
