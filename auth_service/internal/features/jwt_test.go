package features

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
)

func TestJWTServiceRoundTrip(t *testing.T) {
	svc := NewJWTService("test-secret", time.Hour)

	token, err := svc.CreateToken("user-123")
	if err != nil {
		t.Fatalf("CreateToken() error: %v", err)
	}

	id, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error: %v", err)
	}
	if id != "user-123" {
		t.Fatalf("ParseToken() id = %q, want %q", id, "user-123")
	}
}

func TestJWTServiceRejectsWrongSecret(t *testing.T) {
	issuer := NewJWTService("secret-a", time.Hour)
	verifier := NewJWTService("secret-b", time.Hour)

	token, err := issuer.CreateToken("user-123")
	if err != nil {
		t.Fatalf("CreateToken() error: %v", err)
	}
	if _, err := verifier.ParseToken(token); err == nil {
		t.Fatal("ParseToken() expected error for a token signed with another secret")
	}
}

func TestJWTServiceRejectsGarbageToken(t *testing.T) {
	svc := NewJWTService("test-secret", time.Hour)

	if _, err := svc.ParseToken("not-a-jwt"); err == nil {
		t.Fatal("ParseToken() expected error for a malformed token")
	}
}

func TestJWTServiceRequiresSecret(t *testing.T) {
	svc := NewJWTService("", time.Hour)

	if _, err := svc.CreateToken("user-123"); err == nil {
		t.Fatal("CreateToken() expected error when secret is empty")
	}
}

func TestJWTServiceExpires(t *testing.T) {
	svc := NewJWTService("test-secret", time.Hour)

	claims := jwt.MapClaims{
		"ID":  "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("SignedString() error: %v", err)
	}

	if _, err := svc.ParseToken(token); err == nil {
		t.Fatal("ParseToken() expected error for an expired token")
	}
}
