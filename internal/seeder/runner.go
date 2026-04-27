package seeder

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
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS seed_history (name VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`).Error; err != nil {
		return err
	}
	for _, file := range files {
		name := filepath.Base(file)
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM seed_history WHERE name = ?", name).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := sqlDB.Exec(string(content)); err != nil {
			return err
		}
		if err := db.Exec("INSERT INTO seed_history (name) VALUES (?)", name).Error; err != nil {
			return err
		}
	}
	return nil
}
