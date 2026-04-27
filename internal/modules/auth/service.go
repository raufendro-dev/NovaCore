package auth

import (
	"errors"

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
	user := &User{Name: req.Name, Email: req.Email, PasswordHash: hash, Role: "user"}
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
