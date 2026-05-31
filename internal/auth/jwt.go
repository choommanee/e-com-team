package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token is missing, malformed, or expired.
var ErrInvalidToken = errors.New("invalid token")

// TokenManager issues and verifies signed JWTs.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager builds a manager with the given signing secret and token TTL.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// Issue creates a signed token for the given user.
func (m *TokenManager) Issue(userID, email string, now time.Time) (string, error) {
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
		Email: email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(m.secret)
}

// Parse verifies a token and returns the user id and email.
func (m *TokenManager) Parse(tokenStr string) (userID, email string, err error) {
	var c claims
	tok, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return "", "", ErrInvalidToken
	}
	if c.Subject == "" {
		return "", "", ErrInvalidToken
	}
	return c.Subject, c.Email, nil
}
