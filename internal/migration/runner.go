package migration

import (
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

func Run(db *gorm.DB, dir string) error {
	if db == nil {
		return nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := sqlDB.Exec(string(content)); err != nil {
			return err
		}
	}
	return nil
}
