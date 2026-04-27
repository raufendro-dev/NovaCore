package logger

import (
	"github.com/raufendro/novacore/internal/config"
	"go.uber.org/zap"
)

func New(cfg config.AppConfig) (*zap.Logger, error) {
	if cfg.Env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
