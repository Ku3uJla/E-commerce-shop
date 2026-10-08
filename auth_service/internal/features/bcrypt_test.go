package features

import "testing"

func TestBcryptHasherRoundTrip(t *testing.T) {
	hasher := NewBcryptHasher()

	hashed, err := hasher.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	if hashed == "password123" {
		t.Fatal("HashPassword() returned the plaintext password")
	}
	if !hasher.ComparePassword("password123", hashed) {
		t.Fatal("ComparePassword() = false, want true")
	}
	if hasher.ComparePassword("wrong-password", hashed) {
		t.Fatal("ComparePassword() = true for a wrong password, want false")
	}
}
