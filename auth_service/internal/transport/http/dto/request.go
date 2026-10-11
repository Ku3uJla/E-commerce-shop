package dto

// RegisterRequest is the payload accepted by the register endpoint.
// Name is optional and defaults to the local part of the email.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the payload accepted by the login endpoint.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
