package product

import "testing"

func TestNewNameValidation(t *testing.T) {
	if _, err := NewName("  "); err == nil {
		t.Fatal("NewName() expected error for empty name")
	}
	name, err := NewName("  Widget  ")
	if err != nil {
		t.Fatalf("NewName() error: %v", err)
	}
	if name.String() != "Widget" {
		t.Fatalf("NewName().String() = %q, want %q", name.String(), "Widget")
	}
}

func TestNewPriceValidation(t *testing.T) {
	if _, err := NewPrice(-1); err == nil {
		t.Fatal("NewPrice() expected error for negative price")
	}
	price, err := NewPrice(1999)
	if err != nil {
		t.Fatalf("NewPrice() error: %v", err)
	}
	if price.Int64() != 1999 {
		t.Fatalf("NewPrice().Int64() = %d, want 1999", price.Int64())
	}
}

func TestNewStockValidation(t *testing.T) {
	if _, err := NewStock(-5); err == nil {
		t.Fatal("NewStock() expected error for negative stock")
	}
}

func TestNewProductIDValidation(t *testing.T) {
	if _, err := NewProductID("not-a-uuid"); err == nil {
		t.Fatal("NewProductID() expected error for invalid uuid")
	}
	if _, err := NewProductID(GenerateProductID().String()); err != nil {
		t.Fatalf("NewProductID() error: %v", err)
	}
}
