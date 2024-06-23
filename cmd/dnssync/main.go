package main

import (
	"github.com/smirzaei/dnssync/internal/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := initLogger()
	defer func() {
		_ = logger.Sync()
	}()

	args := cli.ParseArgs()

	logger.Info("Hello, world!", zap.Any("args", args))
}

func initLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
