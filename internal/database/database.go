package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/raufendro/novacore/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Connections struct {
	SQL   *gorm.DB
	Mongo *mongo.Database
}

func Connect(cfg *config.Config) (*Connections, error) {
	conn := &Connections{}
	switch cfg.Database.Driver {
	case "mysql":
		if cfg.Database.AutoCreate {
			if err := ensureMySQLDatabase(cfg.Database); err != nil {
				return nil, err
			}
		}
		db, err := gorm.Open(mysql.Open(mysqlDSN(cfg.Database)), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		conn.SQL = db
	case "postgres", "postgresql":
		if cfg.Database.AutoCreate {
			if err := ensurePostgresDatabase(cfg.Database); err != nil {
				return nil, err
			}
		}
		db, err := gorm.Open(postgres.Open(postgresDSN(cfg.Database)), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		conn.SQL = db
	case "mongodb":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
		if err != nil {
			return nil, err
		}
		conn.Mongo = client.Database(cfg.Mongo.Database)
	case "sqlite", "":
		if err := os.MkdirAll(filepath.Dir(cfg.Database.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		db, err := gorm.Open(sqlite.Open(cfg.Database.SQLitePath), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		conn.SQL = db
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.Database.Driver)
	}
	return conn, nil
}

func mysqlDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
}

func mysqlServerDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&charset=utf8mb4&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port)
}

func postgresDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC", cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)
}

func postgresServerDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s TimeZone=UTC", cfg.Host, cfg.User, cfg.Password, cfg.Port, cfg.SSLMode)
}

func ensureMySQLDatabase(cfg config.DatabaseConfig) error {
	name, err := safeDatabaseName(cfg.Name)
	if err != nil {
		return err
	}
	db, err := gorm.Open(mysql.Open(mysqlServerDSN(cfg)), &gorm.Config{})
	if err != nil {
		return err
	}
	return db.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
}

func ensurePostgresDatabase(cfg config.DatabaseConfig) error {
	name, err := safeDatabaseName(cfg.Name)
	if err != nil {
		return err
	}
	db, err := gorm.Open(postgres.Open(postgresServerDSN(cfg)), &gorm.Config{})
	if err != nil {
		return err
	}
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", name).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec(`CREATE DATABASE "` + name + `"`).Error
}

func safeDatabaseName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("DB_NAME is required")
	}
	ok, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("DB_NAME may only contain letters, numbers, and underscores")
	}
	return name, nil
}
