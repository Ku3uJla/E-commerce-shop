package user

import "testing"

func TestNewEmailNormalizesAndValidates(t *testing.T) {
	email, err := NewEmail("  User@Example.COM ")
	if err != nil {
		t.Fatalf("NewEmail() error: %v", err)
	}
	if email.String() != "user@example.com" {
		t.Fatalf("NewEmail().String() = %q, want %q", email.String(), "user@example.com")
	}
}

func TestNewEmailRejectsInvalid(t *testing.T) {
	if _, err := NewEmail("not-an-email"); err == nil {
		t.Fatal("NewEmail() expected error for invalid email")
	}
}

func TestNewNameRejectsEmpty(t *testing.T) {
	if _, err := NewName("   "); err == nil {
		t.Fatal("NewName() expected error for empty name")
	}
}

func TestGenerateUserIDIsValid(t *testing.T) {
	id := GenerateUserID()
	if _, err := NewUserID(id.String()); err != nil {
		t.Fatalf("NewUserID(%q) error: %v", id.String(), err)
	}
}

func TestNewUserIDRejectsInvalid(t *testing.T) {
	if _, err := NewUserID("123"); err == nil {
		t.Fatal("NewUserID() expected error for invalid uuid")
	}
}
