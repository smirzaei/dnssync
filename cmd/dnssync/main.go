package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/smirzaei/dnssync/internal/cli"
	"github.com/smirzaei/dnssync/internal/daemon"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	args := cli.ParseArgs()

	logger := initLogger(args.Verbose)
	defer func() {
		_ = logger.Sync()
	}()

	d, err := daemon.NewDaemon(logger, args)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err := d.Run(ctx)
		if err != nil {
			logger.Error("daemon failure", zap.Error(err))
		}
		logger.Info("daemon is stopped")
		wg.Done()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	s := <-sigChan
	logger.Info("received termination signal, shutting down", zap.String("signal", s.String()))
	cancel()

	wg.Wait()
}

func initLogger(verbose bool) *zap.Logger {
	var logLevel zap.AtomicLevel
	if verbose {
		logLevel = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		logLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.Level = logLevel

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
