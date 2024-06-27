package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/smirzaei/dnssync/internal/cli"
	"github.com/smirzaei/dnssync/internal/daemon"
	"github.com/smirzaei/dnssync/internal/dns"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	args := cli.ParseArgs()

	logger := initLogger(args.Verbose)
	defer func() {
		_ = logger.Sync()
	}()

	if args.Verbose {
		logger.Debug("loaded args", zap.Any("args", args))
	}

	switch strings.ToLower(args.Command) {
	case string(cli.CommandList):
		listDNSRecords(logger, args)
	case string(cli.CommandRun):
		runDaemon(logger, args)
	default:
		logger.Error("unknown command. please use 'run' or 'list'.", zap.String("command", args.Command))
	}
}

func listDNSRecords(logger *zap.Logger, args cli.Args) {
	dnsManager, err := dns.NewDNSManager(logger, args.CloudflareApiKey, args.ZoneID)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	records, err := dnsManager.ListRecords(ctx)
	if err != nil {
		logger.Error("failed to list dns records", zap.Error(err))
		return
	}

	for i, r := range records {
		msg := fmt.Sprintf("%d - %s\t%s\t%s\n", i, r.ID, r.Name, r.Value)
		_, err := os.Stdout.WriteString(msg)
		if err != nil {
			logger.Error("failed to write to stdout", zap.Error(err), zap.String("msg", "msg"))
		}
	}
}

func runDaemon(logger *zap.Logger, args cli.Args) {
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
