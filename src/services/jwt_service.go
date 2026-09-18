package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret     string
	expiration int
	logger     *slog.Logger
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	Menus  string `json:"menus"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, expiration int, logger *slog.Logger) *JWTService {
	return &JWTService{
		secret:     secret,
		expiration: expiration,
		logger:     logger,
	}
}

// GenerateToken creates a new JWT token
func (s *JWTService) GenerateToken(userID uint, email string, role string, menus string) (string, error) {
	claims := Claims{
		Role:   role,
		UserID: userID,
		Email:  email,
		Menus:  menus,
		RegisteredClaims: jwt.RegisteredClaims{
			// expires at is increased by expiration in hours
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.expiration) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tapmymenu",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

// ValidateToken verifies and parses the JWT token
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
