package features

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ComparePassword reports whether the plaintext password matches the bcrypt hash.
func ComparePassword(password, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

// BcryptHasher adapts the package helpers to the domain user.Hash interface.
type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{} }

func (BcryptHasher) HashPassword(password string) (string, error) {
	return HashPassword(password)
}

func (BcryptHasher) ComparePassword(password, hashedPassword string) bool {
	return ComparePassword(password, hashedPassword)
}
