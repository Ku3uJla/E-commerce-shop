package product

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

const (
	maxNameLength        = 150
	maxDescriptionLength = 2000
)

type ProductID struct {
	value string
}

func GenerateProductID() ProductID {
	return ProductID{value: uuid.NewString()}
}

func NewProductID(value string) (ProductID, error) {
	if _, err := uuid.Parse(value); err != nil {
		return ProductID{}, err
	}
	return ProductID{value: value}, nil
}

func (id ProductID) String() string {
	return id.value
}

type Name struct {
	value string
}

func NewName(value string) (Name, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Name{}, errors.New("product name must not be empty")
	}
	if len([]rune(value)) > maxNameLength {
		return Name{}, errors.New("product name is too long")
	}
	return Name{value: value}, nil
}

func (name Name) String() string {
	return name.value
}

type Description struct {
	value string
}

func NewDescription(value string) (Description, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > maxDescriptionLength {
		return Description{}, errors.New("product description is too long")
	}
	return Description{value: value}, nil
}

func (description Description) String() string {
	return description.value
}

// Price is stored in minor units (e.g. cents) to avoid floating point errors.
type Price struct {
	value int64
}

func NewPrice(value int64) (Price, error) {
	if value < 0 {
		return Price{}, errors.New("price must not be negative")
	}
	return Price{value: value}, nil
}

func (price Price) Int64() int64 {
	return price.value
}

type Stock struct {
	value int
}

func NewStock(value int) (Stock, error) {
	if value < 0 {
		return Stock{}, errors.New("stock must not be negative")
	}
	return Stock{value: value}, nil
}

func (stock Stock) Int() int {
	return stock.value
}
