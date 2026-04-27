package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Mongo    MongoConfig
	Redis    RedisConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Rate     RateLimitConfig
	Timeout  time.Duration
}

type AppConfig struct {
	Name  string
	Env   string
	Port  string
	Debug bool
}

type DatabaseConfig struct {
	Driver     string
	Host       string
	Port       string
	Name       string
	User       string
	Password   string
	SSLMode    string
	SQLitePath string
	AutoCreate bool
}

type MongoConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	setDefaults()
	cfg := &Config{
		App: AppConfig{
			Name:  viper.GetString("APP_NAME"),
			Env:   viper.GetString("APP_ENV"),
			Port:  viper.GetString("APP_PORT"),
			Debug: viper.GetBool("APP_DEBUG"),
		},
		Database: DatabaseConfig{
			Driver:     viper.GetString("DB_DRIVER"),
			Host:       viper.GetString("DB_HOST"),
			Port:       viper.GetString("DB_PORT"),
			Name:       viper.GetString("DB_NAME"),
			User:       viper.GetString("DB_USER"),
			Password:   viper.GetString("DB_PASSWORD"),
			SSLMode:    viper.GetString("DB_SSL_MODE"),
			SQLitePath: viper.GetString("DB_SQLITE_PATH"),
			AutoCreate: viper.GetBool("DB_AUTO_CREATE"),
		},
		Mongo: MongoConfig{URI: viper.GetString("MONGO_URI"), Database: viper.GetString("MONGO_DATABASE")},
		Redis: RedisConfig{
			Enabled:  viper.GetBool("REDIS_ENABLED"),
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			AccessExpiry:  viper.GetDuration("JWT_ACCESS_EXPIRY"),
			RefreshExpiry: viper.GetDuration("JWT_REFRESH_EXPIRY"),
		},
		CORS:    CORSConfig{AllowedOrigins: split(viper.GetString("CORS_ALLOWED_ORIGINS"))},
		Rate:    RateLimitConfig{Requests: viper.GetInt("RATE_LIMIT_REQUESTS"), Window: viper.GetDuration("RATE_LIMIT_WINDOW")},
		Timeout: viper.GetDuration("REQUEST_TIMEOUT"),
	}
	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("APP_NAME", "NovaCore")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_DEBUG", true)
	viper.SetDefault("DB_DRIVER", "sqlite")
	viper.SetDefault("DB_SQLITE_PATH", "database/app.db")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("DB_AUTO_CREATE", false)
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGO_DATABASE", "app_db")
	viper.SetDefault("JWT_SECRET", "change_this_secret")
	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "168h")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "*")
	viper.SetDefault("RATE_LIMIT_REQUESTS", 100)
	viper.SetDefault("RATE_LIMIT_WINDOW", "1m")
	viper.SetDefault("REQUEST_TIMEOUT", "15s")
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
