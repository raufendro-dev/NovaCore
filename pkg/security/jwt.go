package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/raufendro/novacore/internal/config"
)

type Claims struct {
	UserID uint     `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Type   string   `json:"type"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(cfg config.JWTConfig) *JWTManager {
	return &JWTManager{secret: []byte(cfg.Secret), accessExpiry: cfg.AccessExpiry, refreshExpiry: cfg.RefreshExpiry}
}

func (m *JWTManager) GeneratePair(userID uint, email string, roles []string) (string, string, error) {
	access, err := m.generate(userID, email, roles, "access", m.accessExpiry)
	if err != nil {
		return "", "", err
	}
	refresh, err := m.generate(userID, email, roles, "refresh", m.refreshExpiry)
	return access, refresh, err
}

func (m *JWTManager) Validate(tokenText string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenText, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (m *JWTManager) generate(userID uint, email string, roles []string, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}
