package features

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

const defaultTokenTTL = 24 * time.Hour

// JWTService signs and verifies JWT access tokens using HMAC-SHA256.
type JWTService struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTService builds a token service from an explicit secret and lifetime.
func NewJWTService(secret string, ttl time.Duration) *JWTService {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &JWTService{secret: []byte(secret), ttl: ttl}
}

// NewJWTServiceFromEnv builds a token service from the JWT_KEY environment variable.
func NewJWTServiceFromEnv() *JWTService {
	return NewJWTService(os.Getenv("JWT_KEY"), defaultTokenTTL)
}

// CreateToken issues a signed token that carries the given user id.
func (s *JWTService) CreateToken(id string) (string, error) {
	if len(s.secret) == 0 {
		return "", errors.New("jwt secret is not configured")
	}
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"ID":  id,
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ParseToken validates a token and returns the user id stored in its claims.
func (s *JWTService) ParseToken(tokenString string) (string, error) {
	if len(s.secret) == 0 {
		return "", errors.New("jwt secret is not configured")
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	id, ok := claims["ID"].(string)
	if !ok || id == "" {
		return "", errors.New("invalid token claims")
	}
	return id, nil
}

// CreateToken issues a token using the JWT_KEY environment variable.
func CreateToken(id string) (string, error) {
	return NewJWTServiceFromEnv().CreateToken(id)
}
