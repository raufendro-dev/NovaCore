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
	Name        string
	Lower       string
	Snake       string
	Kebab       string
	Plural      string
	PluralPath  string
	Public      bool
	Fields      []Field
	Methods     []EndpointMethod
	MethodNames []string
	HasGET      bool
	HasPOST     bool
	HasPUT      bool
	HasPATCH    bool
	HasDELETE   bool
	NeedsTime   bool
	FirstField  *Field
	Relations   []Relation
}

type Options struct {
	Public  bool
	Fields  []Field
	Methods []string
	Mode    string
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
	DefaultValue   string
	SQLDefault     string
	HasDefault     bool
	Required       bool
}

type Relation struct {
	Source           string `json:"source"`
	Target           string `json:"target"`
	Type             string `json:"type"`
	ForeignKey       string `json:"foreign_key"`
	Nested           bool   `json:"nested"`
	Include          bool   `json:"include"`
	UpdatePostman    bool   `json:"update_postman"`
	UpdateDocs       bool   `json:"update_docs"`
	JoinTable        string `json:"join_table,omitempty"`
	SourceSnake      string `json:"-"`
	TargetSnake      string `json:"-"`
	TargetPackage    string `json:"-"`
	TargetField      string `json:"-"`
	TargetFieldMany  string `json:"-"`
	TargetPluralPath string `json:"-"`
	ForeignKeyGo     string `json:"-"`
}

type Metadata struct {
	Name      string     `json:"name"`
	Snake     string     `json:"snake"`
	Plural    string     `json:"plural"`
	Public    bool       `json:"public"`
	Methods   []string   `json:"methods"`
	Fields    []Field    `json:"fields"`
	Relations []Relation `json:"relations"`
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
		if err := SaveMetadata(v); err != nil {
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
	old, _ := LoadMetadata(v.Name)
	v.Relations = hydrateRelations(old.Relations)
	if err := writeCRUD(v); err != nil {
		return err
	}
	if err := SaveMetadata(v); err != nil {
		return err
	}
	if opts.Mode == "safe" {
		return GenerateSafeMigrationForView(old, v)
	}
	return GenerateResetMigrationForView(v)
}

func viewFromOptions(name string, opts Options) View {
	v := makeView(name)
	v.Public = opts.Public
	v.Fields = normalizeFields(opts.Fields)
	v.MethodNames, _ = NormalizeMethodNames(opts.Methods)
	v.Methods = normalizeMethods(v.MethodNames, v.PluralPath)
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

func GenerateSafeMigrationForView(old Metadata, v View) error {
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	base := filepath.Join("migrations", stamp+"_alter_"+v.Plural+"_table")
	up := buildSafeAlterSQL(old, v)
	down := "-- Safe migrations are not automatically reversible. Review changes manually.\n"
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

func ApplyRelation(relation Relation) error {
	if relation.Source == "" || relation.Target == "" {
		return fmt.Errorf("source and target model are required")
	}
	relation.Type = strings.ToLower(strings.TrimSpace(relation.Type))
	switch relation.Type {
	case "belongs-to", "has-one", "has-many", "many-to-many":
	default:
		return fmt.Errorf("unsupported relation type %q", relation.Type)
	}
	sourceMeta, err := LoadMetadata(relation.Source)
	if err != nil {
		return fmt.Errorf("source CRUD metadata not found: %w", err)
	}
	targetMeta, err := LoadMetadata(relation.Target)
	if err != nil {
		return fmt.Errorf("target CRUD metadata not found: %w", err)
	}
	relation.Source = sourceMeta.Name
	relation.Target = targetMeta.Name
	relation = hydrateRelations([]Relation{relation})[0]
	sourceMeta.Relations = upsertRelation(sourceMeta.Relations, relation)
	sourceView := viewFromMetadata(sourceMeta)
	if relation.Type == "belongs-to" {
		fkField, err := NewFieldWithDefault(relation.ForeignKey, "uint", false, "")
		if err != nil {
			return err
		}
		sourceView.Fields = upsertMetadataField(sourceView.Fields, fkField)
		sourceMeta.Fields = sourceView.Fields
	}
	if err := writeCRUD(sourceView); err != nil {
		return err
	}
	if err := SaveMetadata(sourceView); err != nil {
		return err
	}
	if relation.Type == "many-to-many" {
		reverse := Relation{Source: targetMeta.Name, Target: sourceMeta.Name, Type: "many-to-many", Include: relation.Include, UpdateDocs: relation.UpdateDocs, UpdatePostman: relation.UpdatePostman, JoinTable: relation.JoinTable}
		targetMeta.Relations = upsertRelation(targetMeta.Relations, hydrateRelations([]Relation{reverse})[0])
		targetView := viewFromMetadata(targetMeta)
		if err := writeCRUD(targetView); err != nil {
			return err
		}
		if err := SaveMetadata(targetView); err != nil {
			return err
		}
	}
	return GenerateRelationMigration(relation)
}

func GenerateRelationMigration(relation Relation) error {
	if err := os.MkdirAll("migrations", 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102150405")
	base := filepath.Join("migrations", stamp+"_add_"+relation.TargetSnake+"_relation_to_"+relation.SourceSnake)
	var up strings.Builder
	var down strings.Builder
	switch relation.Type {
	case "belongs-to":
		up.WriteString(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s INTEGER;\n", makeView(relation.Source).Plural, relation.ForeignKey))
		down.WriteString(fmt.Sprintf("-- ALTER TABLE %s DROP COLUMN %s;\n", makeView(relation.Source).Plural, relation.ForeignKey))
	case "many-to-many":
		up.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s_id INTEGER NOT NULL,\n  %s_id INTEGER NOT NULL,\n  PRIMARY KEY (%s_id, %s_id)\n);\n", relation.JoinTable, relation.SourceSnake, relation.TargetSnake, relation.SourceSnake, relation.TargetSnake))
		down.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", relation.JoinTable))
	default:
		up.WriteString(fmt.Sprintf("-- %s relation uses existing foreign key. Review schema if needed.\n", relation.Type))
		down.WriteString("-- No automatic rollback.\n")
	}
	if err := os.WriteFile(base+".up.sql", []byte(up.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(base+".down.sql", []byte(down.String()), 0o644)
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
	return NewFieldWithDefault(name, dataType, required, "")
}

func NewFieldWithDefault(name, dataType string, required bool, defaultValue string) (Field, error) {
	snake := toSnake(strings.TrimSpace(name))
	if snake == "" {
		return Field{}, fmt.Errorf("field name is required")
	}
	goType, sqlType, example, jsonExample, zeroValue, err := mapFieldType(dataType)
	if err != nil {
		return Field{}, err
	}
	sqlDefault, hasDefault, err := SQLDefaultLiteral(defaultValue, goType)
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
		DefaultValue:   strings.TrimSpace(defaultValue),
		SQLDefault:     sqlDefault,
		HasDefault:     hasDefault,
		Required:       required,
	}
	if required {
		field.ValidateCreate = "required"
		field.GormTag = "not null"
	}
	if hasDefault {
		if field.GormTag != "" {
			field.GormTag += ";"
		}
		field.GormTag += "default:" + field.DefaultValue
		field.JSONExample = defaultJSONExample(field)
		field.Example = defaultGoExample(field)
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
		normalized, err := NewFieldWithDefault(field.Name, field.GoType, field.Required, field.DefaultValue)
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
		defaultSQL := ""
		if field.HasDefault {
			defaultSQL = " DEFAULT " + field.SQLDefault
		}
		builder.WriteString(fmt.Sprintf("  %s %s%s%s,\n", field.SnakeName, field.SQLType, nullable, defaultSQL))
	}
	builder.WriteString("  created_at TIMESTAMP,\n")
	builder.WriteString("  updated_at TIMESTAMP,\n")
	builder.WriteString("  deleted_at TIMESTAMP\n")
	builder.WriteString(");\n")
	return builder.String()
}

func buildSafeAlterSQL(old Metadata, v View) string {
	oldFields := map[string]Field{}
	for _, field := range old.Fields {
		oldFields[field.SnakeName] = field
	}
	newFields := map[string]Field{}
	for _, field := range v.Fields {
		newFields[field.SnakeName] = field
	}
	var builder strings.Builder
	for _, field := range v.Fields {
		oldField, exists := oldFields[field.SnakeName]
		if !exists {
			nullable := ""
			if field.Required {
				nullable = " NOT NULL"
			}
			defaultSQL := ""
			if field.HasDefault {
				defaultSQL = " DEFAULT " + field.SQLDefault
			}
			builder.WriteString(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s%s%s;\n", v.Plural, field.SnakeName, field.SQLType, nullable, defaultSQL))
			continue
		}
		if oldField.SQLType != field.SQLType || oldField.Required != field.Required || oldField.SQLDefault != field.SQLDefault {
			builder.WriteString(fmt.Sprintf("-- WARNING: Review type/default/nullability change for column %s manually.\n", field.SnakeName))
			builder.WriteString(fmt.Sprintf("-- Old: %s required=%t default=%s\n", oldField.SQLType, oldField.Required, oldField.DefaultValue))
			builder.WriteString(fmt.Sprintf("-- New: %s required=%t default=%s\n", field.SQLType, field.Required, field.DefaultValue))
		}
	}
	for _, field := range old.Fields {
		if _, exists := newFields[field.SnakeName]; !exists {
			builder.WriteString(fmt.Sprintf("-- WARNING: Column %s was removed from the model. Dropping columns is database-specific and can destroy data.\n", field.SnakeName))
			builder.WriteString(fmt.Sprintf("-- ALTER TABLE %s DROP COLUMN %s;\n", v.Plural, field.SnakeName))
		}
	}
	if builder.Len() == 0 {
		builder.WriteString("-- No schema changes detected.\n")
	}
	return builder.String()
}

func SQLDefaultLiteral(value, goType string) (string, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, nil
	}
	switch goType {
	case "string":
		return "'" + strings.ReplaceAll(value, "'", "''") + "'", true, nil
	case "int", "uint":
		for _, r := range value {
			if r < '0' || r > '9' {
				return "", false, fmt.Errorf("default value for %s must be an integer", goType)
			}
		}
		return value, true, nil
	case "float64":
		if _, err := fmt.Sscan(value, new(float64)); err != nil {
			return "", false, fmt.Errorf("default value for float must be numeric")
		}
		return value, true, nil
	case "bool":
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" {
			return "", false, fmt.Errorf("default value for bool must be true or false")
		}
		return lower, true, nil
	case "time.Time":
		upper := strings.ToUpper(value)
		if upper == "CURRENT_TIMESTAMP" || upper == "NOW()" {
			return upper, true, nil
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return "", false, fmt.Errorf("default value for time must be RFC3339, CURRENT_TIMESTAMP, or NOW()")
		}
		return "'" + strings.ReplaceAll(value, "'", "''") + "'", true, nil
	default:
		return "", false, fmt.Errorf("unsupported default type %s", goType)
	}
}

func defaultJSONExample(field Field) string {
	switch field.GoType {
	case "string", "time.Time":
		if strings.HasPrefix(field.SQLDefault, "'") {
			return `"` + strings.Trim(field.SQLDefault, "'") + `"`
		}
		return `"` + field.DefaultValue + `"`
	default:
		return field.DefaultValue
	}
}

func defaultGoExample(field Field) string {
	switch field.GoType {
	case "string":
		return `"` + strings.ReplaceAll(field.DefaultValue, `"`, `\"`) + `"`
	case "time.Time":
		return `time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)`
	default:
		return field.DefaultValue
	}
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

func SaveMetadata(v View) error {
	meta := Metadata{
		Name:      v.Name,
		Snake:     v.Snake,
		Plural:    v.Plural,
		Public:    v.Public,
		Methods:   v.MethodNames,
		Fields:    v.Fields,
		Relations: v.Relations,
	}
	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(moduleDir(v), "novacore.json"), bytes, 0o644)
}

func LoadMetadata(name string) (Metadata, error) {
	v := makeView(name)
	content, err := os.ReadFile(filepath.Join(moduleDir(v), "novacore.json"))
	if err != nil {
		return Metadata{Name: v.Name, Snake: v.Snake, Plural: v.Plural, Methods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"}}, err
	}
	var meta Metadata
	if err := json.Unmarshal(content, &meta); err != nil {
		return meta, err
	}
	meta.Relations = hydrateRelations(meta.Relations)
	return meta, nil
}

func hydrateRelations(relations []Relation) []Relation {
	out := make([]Relation, 0, len(relations))
	for _, relation := range relations {
		source := makeView(relation.Source)
		target := makeView(relation.Target)
		relation.SourceSnake = source.Snake
		relation.TargetSnake = target.Snake
		relation.TargetPackage = target.Snake
		relation.TargetField = target.Name
		relation.TargetPluralPath = target.PluralPath
		relation.TargetFieldMany = target.Name
		if strings.HasSuffix(relation.TargetFieldMany, "y") {
			relation.TargetFieldMany = strings.TrimSuffix(relation.TargetFieldMany, "y") + "ies"
		} else {
			relation.TargetFieldMany += "s"
		}
		if relation.ForeignKey == "" {
			relation.ForeignKey = target.Snake + "_id"
		}
		relation.ForeignKeyGo = toPascal(relation.ForeignKey)
		if relation.JoinTable == "" && relation.Type == "many-to-many" {
			relation.JoinTable = source.Snake + "_" + target.Plural
		}
		out = append(out, relation)
	}
	return out
}

func viewFromMetadata(meta Metadata) View {
	v := viewFromOptions(meta.Name, Options{Public: meta.Public, Fields: meta.Fields, Methods: meta.Methods})
	v.Relations = hydrateRelations(meta.Relations)
	return v
}

func upsertRelation(relations []Relation, relation Relation) []Relation {
	for i := range relations {
		if relations[i].Target == relation.Target && relations[i].Type == relation.Type {
			relations[i] = relation
			return hydrateRelations(relations)
		}
	}
	return hydrateRelations(append(relations, relation))
}

func upsertMetadataField(fields []Field, field Field) []Field {
	for i := range fields {
		if fields[i].SnakeName == field.SnakeName {
			fields[i] = field
			return fields
		}
	}
	return append(fields, field)
}
