package auth

import (
	"testing"
	"time"

	"github.com/raufendro/novacore/internal/config"
	"github.com/raufendro/novacore/pkg/security"
)

type memoryRepo struct {
	users []User
}

func (m *memoryRepo) Create(user *User) error {
	user.ID = uint(len(m.users) + 1)
	m.users = append(m.users, *user)
	return nil
}

func (m *memoryRepo) FindByEmail(email string) (*User, error) {
	for i := range m.users {
		if m.users[i].Email == email {
			return &m.users[i], nil
		}
	}
	return nil, nil
}

func (m *memoryRepo) FindByID(id uint) (*User, error) {
	for i := range m.users {
		if m.users[i].ID == id {
			return &m.users[i], nil
		}
	}
	return nil, nil
}

func TestRegisterAndLogin(t *testing.T) {
	service := NewService(&memoryRepo{}, testJWT())
	registered, err := service.Register(RegisterRequest{Name: "Demo", Email: "demo@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.Role != "user" {
		t.Fatalf("expected default role user, got %s", registered.User.Role)
	}
	resp, err := service.Login(LoginRequest{Email: "demo@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected token pair")
	}
}

func TestRegisterWithRole(t *testing.T) {
	service := NewService(&memoryRepo{}, testJWT())
	resp, err := service.Register(RegisterRequest{Name: "Admin", Email: "admin@example.com", Password: "password123", Role: "Admin"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.User.Role != "admin" {
		t.Fatalf("expected normalized role admin, got %s", resp.User.Role)
	}
}

func TestRegisterRejectsInvalidRole(t *testing.T) {
	service := NewService(&memoryRepo{}, testJWT())
	_, err := service.Register(RegisterRequest{Name: "Demo", Email: "demo@example.com", Password: "password123", Role: "super admin"})
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func testJWT() *security.JWTManager {
	return security.NewJWTManager(config.JWTConfig{Secret: "test_secret", AccessExpiry: time.Minute, RefreshExpiry: time.Hour})
}
