package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"
)

type View struct {
	Name       string
	Lower      string
	Snake      string
	Kebab      string
	Plural     string
	PluralPath string
	Public     bool
}

type Options struct {
	Public bool
}

func Generate(kind, name string) error {
	return GenerateWithOptions(kind, name, Options{})
}

func GenerateWithOptions(kind, name string, opts Options) error {
	v := makeView(name)
	v.Public = opts.Public
	if kind == "crud" || kind == "module" {
		if err := os.MkdirAll(moduleDir(v), 0o755); err != nil {
			return err
		}
	}
	steps := map[string]func(View) error{
		"model":      writeModel,
		"controller": writeHandler,
		"service":    writeService,
		"repository": writeRepository,
		"endpoint":   writeRoutes,
	}
	if kind == "crud" || kind == "module" {
		for _, fn := range []func(View) error{writeModel, writeDTO, writeRepository, writeService, writeHandler, writeRoutes, writeTest} {
			if err := fn(v); err != nil {
				return err
			}
		}
		if err := GenerateMigration("create_" + v.Plural + "_table"); err != nil {
			return err
		}
		if err := updateRouteRegistry(v); err != nil {
			return err
		}
		if err := UpsertPostman(v); err != nil {
			return err
		}
		return writeEndpointDoc(v)
	}
	fn, ok := steps[kind]
	if !ok {
		return fmt.Errorf("unknown generator kind %q", kind)
	}
	return fn(v)
}

func GenerateMigration(name string) error {
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	base := filepath.Join("migrations", stamp+"_"+toSnake(name))
	up := "-- Write your SQL migration here.\n"
	down := "-- Write your rollback SQL here.\n"
	if strings.HasPrefix(name, "create_") && strings.HasSuffix(name, "_table") {
		table := strings.TrimSuffix(strings.TrimPrefix(name, "create_"), "_table")
		up = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  id INTEGER PRIMARY KEY,\n  name VARCHAR(120) NOT NULL,\n  created_at TIMESTAMP,\n  updated_at TIMESTAMP,\n  deleted_at TIMESTAMP\n);\n", table)
		down = fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", table)
	}
	if err := os.WriteFile(base+".up.sql", []byte(up), 0o644); err != nil {
		return err
	}
	return os.WriteFile(base+".down.sql", []byte(down), 0o644)
}

func GenerateSeeder(name string) error {
	if err := os.MkdirAll("seeders", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	path := filepath.Join("seeders", stamp+"_"+toSnake(name)+".sql")
	body := "-- Write idempotent seed SQL here.\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

func writeTemplate(path, body string, v View) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	t, err := template.New(path).Parse(body)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func moduleDir(v View) string { return filepath.Join("internal", "modules", v.Snake) }

func makeView(name string) View {
	clean := strings.TrimSpace(name)
	clean = strings.ToUpper(clean[:1]) + clean[1:]
	snake := toSnake(clean)
	plural := pluralize(snake)
	return View{Name: clean, Lower: strings.ToLower(clean[:1]) + clean[1:], Snake: snake, Kebab: strings.ReplaceAll(snake, "_", "-"), Plural: plural, PluralPath: strings.ReplaceAll(plural, "_", "-")}
}

func toSnake(value string) string {
	var out []rune
	for i, r := range value {
		if unicode.IsUpper(r) && i > 0 {
			out = append(out, '_')
		}
		if r == '-' || r == ' ' {
			out = append(out, '_')
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

func pluralize(value string) string {
	if strings.HasSuffix(value, "y") {
		return strings.TrimSuffix(value, "y") + "ies"
	}
	if strings.HasSuffix(value, "s") {
		return value + "es"
	}
	return value + "s"
}

func updateRouteRegistry(v View) error {
	path := filepath.Join("internal", "routes", "generated.go")
	entries, err := generatedModules()
	if err != nil {
		return err
	}
	var imports strings.Builder
	var calls strings.Builder
	for _, entry := range entries {
		imports.WriteString(fmt.Sprintf("\t%s \"github.com/raufendro/novacore/internal/modules/%s\"\n", entry, entry))
		calls.WriteString(fmt.Sprintf("\t%s.RegisterRoutes(router, db, jwt)\n", entry))
	}
	body := fmt.Sprintf(`package routes

import (
	"github.com/gin-gonic/gin"
%s
	"github.com/raufendro/novacore/pkg/security"
	"gorm.io/gorm"
)

func RegisterGenerated(router *gin.RouterGroup, db *gorm.DB, jwt *security.JWTManager) {
%s
}
`, imports.String(), calls.String())
	return os.WriteFile(path, []byte(body), 0o644)
}

func generatedModules() ([]string, error) {
	dirs, err := os.ReadDir(filepath.Join("internal", "modules"))
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, dir := range dirs {
		if !dir.IsDir() || dir.Name() == "auth" || dir.Name() == "health" {
			continue
		}
		if _, err := os.Stat(filepath.Join("internal", "modules", dir.Name(), "routes.go")); err == nil {
			out = append(out, dir.Name())
		}
	}
	return out, nil
}

func writeEndpointDoc(v View) error {
	path := filepath.Join("docs", "endpoints_"+v.Snake+".md")
	body := fmt.Sprintf("# %s Endpoints\n\nBase path: `/api/v1/%s`\n\n- `GET /api/v1/%s`\n- `GET /api/v1/%s/:id`\n- `POST /api/v1/%s`\n- `PUT /api/v1/%s/:id`\n- `PATCH /api/v1/%s/:id`\n- `DELETE /api/v1/%s/:id`\n", v.Name, v.PluralPath, v.PluralPath, v.PluralPath, v.PluralPath, v.PluralPath, v.PluralPath, v.PluralPath)
	return os.WriteFile(path, []byte(body), 0o644)
}

func writeModel(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "model.go"), modelTpl, v)
}
func writeDTO(v View) error { return writeTemplate(filepath.Join(moduleDir(v), "dto.go"), dtoTpl, v) }
func writeRepository(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "repository.go"), repositoryTpl, v)
}
func writeService(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "service.go"), serviceTpl, v)
}
func writeHandler(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "handler.go"), handlerTpl, v)
}
func writeRoutes(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "routes.go"), routesTpl, v)
}
func writeTest(v View) error {
	return writeTemplate(filepath.Join(moduleDir(v), "service_test.go"), testTpl, v)
}

func UpsertPostman(v View) error {
	if err := os.MkdirAll("postman", 0o755); err != nil {
		return err
	}
	collection := map[string]any{
		"info": map[string]any{"name": "NovaCore API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		"item": []any{authFolder(), moduleFolder(v)},
	}
	bytes, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("postman/collection.json", bytes, 0o644); err != nil {
		return err
	}
	env := map[string]any{"name": "NovaCore Local", "values": []any{
		map[string]any{"key": "base_url", "value": "http://localhost:8080/api/v1", "enabled": true},
		map[string]any{"key": "token", "value": "", "enabled": true},
	}}
	envBytes, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("postman/environment.json", envBytes, 0o644)
}

func authFolder() map[string]any {
	return map[string]any{"name": "Auth", "item": []any{
		map[string]any{"name": "Register", "request": map[string]any{"method": "POST", "header": []any{map[string]any{"key": "Content-Type", "value": "application/json"}}, "url": "{{base_url}}/auth/register", "body": map[string]any{"mode": "raw", "raw": `{"name":"Demo User","email":"demo@example.com","password":"password123"}`}}},
		map[string]any{"name": "Login", "request": map[string]any{"method": "POST", "header": []any{map[string]any{"key": "Content-Type", "value": "application/json"}}, "url": "{{base_url}}/auth/login", "body": map[string]any{"mode": "raw", "raw": `{"email":"demo@example.com","password":"password123"}`}}},
		map[string]any{"name": "Me", "request": map[string]any{"method": "GET", "header": []any{map[string]any{"key": "Authorization", "value": "Bearer {{token}}"}}, "url": "{{base_url}}/auth/me"}},
	}}
}

func moduleFolder(v View) map[string]any {
	methods := []string{"GET", "GET", "POST", "PUT", "PATCH", "DELETE"}
	paths := []string{v.PluralPath, v.PluralPath + "/:id", v.PluralPath, v.PluralPath + "/:id", v.PluralPath + "/:id", v.PluralPath + "/:id"}
	items := make([]any, 0, len(methods))
	for i, method := range methods {
		items = append(items, map[string]any{
			"name": method + " /" + paths[i],
			"request": map[string]any{
				"method": method,
				"header": []any{map[string]any{"key": "Authorization", "value": "Bearer {{token}}"}, map[string]any{"key": "Content-Type", "value": "application/json"}},
				"url":    map[string]any{"raw": "{{base_url}}/" + paths[i], "host": []string{"{{base_url}}"}, "path": strings.Split(paths[i], "/")},
				"body":   map[string]any{"mode": "raw", "raw": `{"name":"Example"}`},
			},
			"response": []any{map[string]any{"name": "Success", "body": `{"success":true,"message":"OK","data":{}}`}},
		})
	}
	return map[string]any{"name": v.Name, "item": items}
}
