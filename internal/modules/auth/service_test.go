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
	_, err := service.Register(RegisterRequest{Name: "Demo", Email: "demo@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := service.Login(LoginRequest{Email: "demo@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected token pair")
	}
}

func testJWT() *security.JWTManager {
	return security.NewJWTManager(config.JWTConfig{Secret: "test_secret", AccessExpiry: time.Minute, RefreshExpiry: time.Hour})
}
