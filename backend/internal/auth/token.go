package auth

import (
	"fmt"
	"time"

	"aegis/pkg/authctx"
	"aegis/pkg/database"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type Tokens struct {
	Secret    []byte
	AccessTTL time.Duration
	Now       func() time.Time
}

func (t Tokens) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func (t Tokens) IssueAccess(user database.User) (string, error) {
	now := t.now()
	claims := Claims{
		Role:  string(user.Role),
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.Secret)
}

func (t Tokens) ParseAccess(raw string) (authctx.Principal, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.Secret, nil
	}, jwt.WithTimeFunc(t.now))
	if err != nil {
		return authctx.Principal{}, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return authctx.Principal{}, fmt.Errorf("invalid token")
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return authctx.Principal{}, err
	}
	return authctx.Principal{
		ID:    id,
		Role:  database.Role(claims.Role),
		Email: claims.Email,
	}, nil
}
