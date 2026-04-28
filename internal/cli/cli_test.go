package cli

import (
	"strings"
	"testing"
)

func TestPatchAuthDTOAddsRole(t *testing.T) {
	input := "type RegisterRequest struct {\n\tName     string `json:\"name\" validate:\"required,min=2,max=120\"`\n\tEmail    string `json:\"email\" validate:\"required,email\"`\n\tPassword string `json:\"password\" validate:\"required,min=8\"`\n}\n"
	got := patchAuthDTO(input)
	if !containsAll(got, "Role     string `json:\"role\" validate:\"omitempty,min=2,max=40\"`") {
		t.Fatalf("expected role field, got:\n%s", got)
	}
}

func TestPatchAuthServiceAddsRoleNormalization(t *testing.T) {
	input := "package auth\n\nimport (\n\t\"errors\"\n\n\t\"example.com/app/pkg/security\"\n)\n\nfunc (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {\n\thash, err := security.HashPassword(req.Password)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tuser := &User{Name: req.Name, Email: req.Email, PasswordHash: hash}\n\treturn s.tokens(user)\n}\n"
	got := patchAuthService(input)
	if !containsAll(got, "\"strings\"", "\"unicode\"", "normalizeRole(req.Role)", "func normalizeRole(role string) (string, error)") {
		t.Fatalf("expected auth service role patch, got:\n%s", got)
	}
}

func TestPatchAuthModelAddsRole(t *testing.T) {
	input := "type User struct {\n\tgorm.Model\n\tName         string `json:\"name\" gorm:\"size:120;not null\"`\n\tEmail        string `json:\"email\" gorm:\"size:180;uniqueIndex;not null\"`\n\tPasswordHash string `json:\"-\" gorm:\"not null\"`\n}\n"
	got := patchAuthModel(input)
	if !containsAll(got, "Role         string `json:\"role\" gorm:\"size:40;not null;default:user\"`", "func (u User) Roles() []string") {
		t.Fatalf("expected auth model role patch, got:\n%s", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
