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
	Fields     []Field
	Methods    []EndpointMethod
	HasGET     bool
	HasPOST    bool
	HasPUT     bool
	HasPATCH   bool
	HasDELETE  bool
	NeedsTime  bool
	FirstField *Field
}

type Options struct {
	Public  bool
	Fields  []Field
	Methods []string
}

type EndpointMethod struct {
	Method string
	Path   string
}

type Field struct {
	Name           string
	GoName         string
	JSONName       string
	SnakeName      string
	GoType         string
	SQLType        string
	GormTag        string
	ValidateCreate string
	ValidateUpdate string
	Example        string
	JSONExample    string
	ZeroValue      string
	Required       bool
}

func Generate(kind, name string) error {
	return GenerateWithOptions(kind, name, Options{})
}

func GenerateWithOptions(kind, name string, opts Options) error {
	v := viewFromOptions(name, opts)
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
		if err := writeCRUD(v); err != nil {
			return err
		}
		return GenerateMigrationForView(v)
	}
	fn, ok := steps[kind]
	if !ok {
		return fmt.Errorf("unknown generator kind %q", kind)
	}
	return fn(v)
}

func UpdateCRUD(name string, opts Options) error {
	v := viewFromOptions(name, opts)
	if _, err := os.Stat(moduleDir(v)); err != nil {
		return fmt.Errorf("module %s does not exist; run make:crud %s first", v.Snake, v.Name)
	}
	if err := writeCRUD(v); err != nil {
		return err
	}
	return GenerateResetMigrationForView(v)
}

func viewFromOptions(name string, opts Options) View {
	v := makeView(name)
	v.Public = opts.Public
	v.Fields = normalizeFields(opts.Fields)
	v.Methods = normalizeMethods(opts.Methods, v.PluralPath)
	v.HasGET = hasMethod(v.Methods, "GET")
	v.HasPOST = hasMethod(v.Methods, "POST")
	v.HasPUT = hasMethod(v.Methods, "PUT")
	v.HasPATCH = hasMethod(v.Methods, "PATCH")
	v.HasDELETE = hasMethod(v.Methods, "DELETE")
	v.NeedsTime = needsTime(v.Fields)
	if len(v.Fields) > 0 {
		v.FirstField = &v.Fields[0]
	}
	return v
}

func writeCRUD(v View) error {
	for _, fn := range []func(View) error{writeModel, writeDTO, writeRepository, writeService, writeHandler, writeRoutes, writeTest} {
		if err := fn(v); err != nil {
			return err
		}
	}
	if err := updateRouteRegistry(v); err != nil {
		return err
	}
	if err := UpsertPostman(v); err != nil {
		return err
	}
	return writeEndpointDoc(v)
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

func GenerateMigrationForView(v View) error {
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	base := filepath.Join("migrations", stamp+"_create_"+v.Plural+"_table")
	up := buildCreateTableSQL(v)
	down := fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", v.Plural)
	if err := os.WriteFile(base+".up.sql", []byte(up), 0o644); err != nil {
		return err
	}
	return os.WriteFile(base+".down.sql", []byte(down), 0o644)
}

func GenerateResetMigrationForView(v View) error {
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	base := filepath.Join("migrations", stamp+"_reset_"+v.Plural+"_table")
	up := fmt.Sprintf("DROP TABLE IF EXISTS %s;\n%s", v.Plural, buildCreateTableSQL(v))
	down := fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", v.Plural)
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

func NewField(name, dataType string, required bool) (Field, error) {
	snake := toSnake(strings.TrimSpace(name))
	if snake == "" {
		return Field{}, fmt.Errorf("field name is required")
	}
	goType, sqlType, example, jsonExample, zeroValue, err := mapFieldType(dataType)
	if err != nil {
		return Field{}, err
	}
	field := Field{
		Name:           snake,
		GoName:         toPascal(snake),
		JSONName:       snake,
		SnakeName:      snake,
		GoType:         goType,
		SQLType:        sqlType,
		ValidateCreate: "omitempty",
		ValidateUpdate: "omitempty",
		Example:        example,
		JSONExample:    jsonExample,
		ZeroValue:      zeroValue,
		Required:       required,
	}
	if required {
		field.ValidateCreate = "required"
		field.GormTag = "not null"
	}
	if goType == "string" && required {
		field.ValidateCreate = "required,min=1"
	}
	return field, nil
}

func normalizeFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		if field.SQLType != "" {
			out = append(out, field)
			continue
		}
		normalized, err := NewField(field.Name, field.GoType, field.Required)
		if err == nil {
			out = append(out, normalized)
		}
	}
	return out
}

func NormalizeMethodNames(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE"}, nil
	}
	allowed := map[string]struct{}{"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}}
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		method := strings.ToUpper(strings.TrimSpace(value))
		if method == "" {
			continue
		}
		if _, ok := allowed[method]; !ok {
			return nil, fmt.Errorf("unsupported method %q", method)
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one method is required")
	}
	return out, nil
}

func normalizeMethods(values []string, pluralPath string) []EndpointMethod {
	names, err := NormalizeMethodNames(values)
	if err != nil {
		names = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	}
	out := []EndpointMethod{}
	for _, name := range names {
		switch name {
		case "GET":
			out = append(out, EndpointMethod{Method: "GET", Path: pluralPath})
			out = append(out, EndpointMethod{Method: "GET", Path: pluralPath + "/:id"})
		case "POST":
			out = append(out, EndpointMethod{Method: "POST", Path: pluralPath})
		case "PUT":
			out = append(out, EndpointMethod{Method: "PUT", Path: pluralPath + "/:id"})
		case "PATCH":
			out = append(out, EndpointMethod{Method: "PATCH", Path: pluralPath + "/:id"})
		case "DELETE":
			out = append(out, EndpointMethod{Method: "DELETE", Path: pluralPath + "/:id"})
		}
	}
	return out
}

func hasMethod(methods []EndpointMethod, method string) bool {
	for _, item := range methods {
		if item.Method == method {
			return true
		}
	}
	return false
}

func mapFieldType(dataType string) (string, string, string, string, string, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "string", "":
		return "string", "VARCHAR(255)", `"Example"`, `"Example"`, `""`, nil
	case "text":
		return "string", "TEXT", `"Long text"`, `"Long text"`, `""`, nil
	case "int", "integer":
		return "int", "INTEGER", "100", "100", "0", nil
	case "uint":
		return "uint", "INTEGER", "100", "100", "0", nil
	case "float", "decimal", "double":
		return "float64", "DECIMAL(12,2)", "99.5", "99.5", "0", nil
	case "bool", "boolean":
		return "bool", "BOOLEAN", "true", "true", "false", nil
	case "time", "datetime", "timestamp":
		return "time.Time", "TIMESTAMP", `time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)`, `"2026-04-27T00:00:00Z"`, "time.Time{}", nil
	default:
		return "", "", "", "", "", fmt.Errorf("unsupported field type %q", dataType)
	}
}

func needsTime(fields []Field) bool {
	for _, field := range fields {
		if field.GoType == "time.Time" {
			return true
		}
	}
	return false
}

func toPascal(value string) string {
	parts := strings.Split(toSnake(value), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func buildCreateTableSQL(v View) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", v.Plural))
	builder.WriteString("  id INTEGER PRIMARY KEY,\n")
	for _, field := range v.Fields {
		nullable := ""
		if field.Required {
			nullable = " NOT NULL"
		}
		builder.WriteString(fmt.Sprintf("  %s %s%s,\n", field.SnakeName, field.SQLType, nullable))
	}
	builder.WriteString("  created_at TIMESTAMP,\n")
	builder.WriteString("  updated_at TIMESTAMP,\n")
	builder.WriteString("  deleted_at TIMESTAMP\n")
	builder.WriteString(");\n")
	return builder.String()
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
	body := fmt.Sprintf("# %s Endpoints\n\nBase path: `/api/v1/%s`\n\n%s\n## Fields\n\n%s", v.Name, v.PluralPath, endpointDocs(v), fieldDocs(v))
	return os.WriteFile(path, []byte(body), 0o644)
}

func endpointDocs(v View) string {
	var builder strings.Builder
	for _, method := range v.Methods {
		builder.WriteString(fmt.Sprintf("- `%s /api/v1/%s`\n", method.Method, method.Path))
	}
	return builder.String()
}

func fieldDocs(v View) string {
	if len(v.Fields) == 0 {
		return "This module has no custom fields beyond `id`, `created_at`, `updated_at`, and `deleted_at`.\n"
	}
	var builder strings.Builder
	builder.WriteString("| Field | Type | Required |\n| --- | --- | --- |\n")
	for _, field := range v.Fields {
		builder.WriteString(fmt.Sprintf("| `%s` | `%s` | `%t` |\n", field.JSONName, field.GoType, field.Required))
	}
	return builder.String()
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
	items := make([]any, 0, len(v.Methods))
	for _, method := range v.Methods {
		items = append(items, map[string]any{
			"name": method.Method + " /" + method.Path,
			"request": map[string]any{
				"method": method.Method,
				"header": []any{map[string]any{"key": "Authorization", "value": "Bearer {{token}}"}, map[string]any{"key": "Content-Type", "value": "application/json"}},
				"url":    map[string]any{"raw": "{{base_url}}/" + method.Path, "host": []string{"{{base_url}}"}, "path": strings.Split(method.Path, "/")},
				"body":   map[string]any{"mode": "raw", "raw": bodyExample(v)},
			},
			"response": []any{map[string]any{"name": "Success", "body": `{"success":true,"message":"OK","data":{}}`}},
		})
	}
	return map[string]any{"name": v.Name, "item": items}
}

func bodyExample(v View) string {
	if len(v.Fields) == 0 {
		return `{}`
	}
	parts := make([]string, 0, len(v.Fields))
	for _, field := range v.Fields {
		parts = append(parts, fmt.Sprintf(`"%s":%s`, field.JSONName, field.JSONExample))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
