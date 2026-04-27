package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldDefaultValueValidation(t *testing.T) {
	field, err := NewFieldWithDefault("stock", "int", true, "0")
	if err != nil {
		t.Fatal(err)
	}
	if !field.HasDefault || field.SQLDefault != "0" {
		t.Fatalf("expected SQL default 0, got %#v", field)
	}
	if _, err := NewFieldWithDefault("stock", "int", true, "abc"); err == nil {
		t.Fatal("expected invalid int default to fail")
	}
}

func TestSafeMigrationUsesAlterTable(t *testing.T) {
	dir := t.TempDir()
	oldWD, _ := os.Getwd()
	defer os.Chdir(oldWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	old := Metadata{Fields: []Field{mustField(t, "name", "string", true, "")}}
	view := viewFromOptions("Product", Options{
		Fields: []Field{
			mustField(t, "name", "string", true, ""),
			mustField(t, "stock", "int", true, "0"),
		},
		Methods: []string{"GET", "POST"},
	})
	if err := GenerateSafeMigrationForView(old, view); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join("migrations", "*_alter_products_table.up.sql"))
	if err != nil || len(files) != 1 {
		t.Fatalf("expected safe migration file, files=%v err=%v", files, err)
	}
	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	sql := string(content)
	if strings.Contains(sql, "DROP TABLE") {
		t.Fatalf("safe migration must not drop table: %s", sql)
	}
	if !strings.Contains(sql, "ALTER TABLE products ADD COLUMN stock INTEGER NOT NULL DEFAULT 0;") {
		t.Fatalf("missing add column SQL: %s", sql)
	}
}

func TestRelationGeneratorCreatesMigration(t *testing.T) {
	dir := t.TempDir()
	oldWD, _ := os.Getwd()
	defer os.Chdir(oldWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join("internal", "routes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := GenerateWithOptions("crud", "Category", Options{Fields: []Field{mustField(t, "name", "string", true, "")}}); err != nil {
		t.Fatal(err)
	}
	if err := GenerateWithOptions("crud", "Product", Options{Fields: []Field{mustField(t, "name", "string", true, "")}}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRelation(Relation{Source: "Product", Target: "Category", Type: "belongs-to", Include: true}); err != nil {
		t.Fatal(err)
	}
	meta, err := LoadMetadata("Product")
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Relations) != 1 || meta.Relations[0].Type != "belongs-to" {
		t.Fatalf("expected belongs-to metadata, got %#v", meta.Relations)
	}
	files, _ := filepath.Glob(filepath.Join("migrations", "*_add_category_relation_to_product.up.sql"))
	if len(files) != 1 {
		t.Fatalf("expected relation migration, got %v", files)
	}
}

func mustField(t *testing.T, name, dataType string, required bool, def string) Field {
	t.Helper()
	field, err := NewFieldWithDefault(name, dataType, required, def)
	if err != nil {
		t.Fatal(err)
	}
	return field
}
