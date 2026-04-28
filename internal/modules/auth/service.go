package auth

import (
	"errors"
	"strings"
	"unicode"

	"github.com/raufendro/novacore/pkg/security"
)

type Service struct {
	repo Repository
	jwt  *security.JWTManager
}

func NewService(repo Repository, jwt *security.JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	existing, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	role, err := normalizeRole(req.Role)
	if err != nil {
		return nil, err
	}
	user := &User{Name: req.Name, Email: req.Email, PasswordHash: hash, Role: role}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return s.tokens(user)
}

func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || !security.CheckPassword(user.PasswordHash, req.Password) {
		return nil, errors.New("invalid credentials")
	}
	return s.tokens(user)
}

func (s *Service) Refresh(refreshToken string) (*AuthResponse, error) {
	claims, err := s.jwt.Validate(refreshToken)
	if err != nil || claims.Type != "refresh" {
		return nil, errors.New("invalid refresh token")
	}
	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return s.tokens(user)
}

func (s *Service) Current(id uint) (*User, error) {
	return s.repo.FindByID(id)
}

func (s *Service) tokens(user *User) (*AuthResponse, error) {
	access, refresh, err := s.jwt.GeneratePair(user.ID, user.Email, user.Roles())
	if err != nil {
		return nil, err
	}
	return &AuthResponse{User: *user, AccessToken: access, RefreshToken: refresh}, nil
}

func normalizeRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "user", nil
	}
	for _, r := range role {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return "", errors.New("role may only contain letters, numbers, underscore, or dash")
	}
	return role, nil
}
