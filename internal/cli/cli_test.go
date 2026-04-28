package cli

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestFindCurrentGoProjectRootPrefersNearestGoMod(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "apps", "demo")
	nested := filepath.Join(child, "internal", "modules")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	got, err := findCurrentGoProjectRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(child)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected nearest Go project root %s, got %s", child, got)
	}
}

func TestCheckCommandPrintsCurrentProjectDetails(t *testing.T) {
	project := t.TempDir()
	serverDir := filepath.Join(project, "cmd", "server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(project, "go.mod"):       "module demo\n",
		filepath.Join(serverDir, "main.go"):    "package main\n",
		filepath.Join(project, ".env"):         "APP_PORT=8080\n",
		filepath.Join(project, ".env.example"): "APP_PORT=8080\n",
	}
	if err := os.MkdirAll(filepath.Join(project, "postman"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, "postman", "environment.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cmd := checkCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	wantProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(got, "NovaCore directory check", "Project root", wantProject, "Go module         : demo", "Server entrypoint", "(found)", "Run command") {
		t.Fatalf("unexpected check output:\n%s", got)
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
