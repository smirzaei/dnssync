package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type HTTPServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type MetricsProvider interface {
	GetCollectors() []prometheus.Collector
}

type HTTPServer struct {
	conf   HTTPServerConfig
	server *http.Server
	logger *zap.Logger
}

func NewHTTPServer(logger *zap.Logger, conf HTTPServerConfig, metricsProvider MetricsProvider) *HTTPServer {
	mux := http.NewServeMux()
	registry := prometheus.NewRegistry()

	collectors := metricsProvider.GetCollectors()
	if len(collectors) > 0 {
		registry.MustRegister(collectors...)
	}

	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog: zap.NewStdLog(logger.Named("prom-http")),
	})
	mux.Handle("/metrics", metricsHandler)

	if conf.ReadTimeout == 0 {
		conf.ReadTimeout = 5 * time.Second
	}
	if conf.WriteTimeout == 0 {
		conf.WriteTimeout = 5 * time.Second
	}
	if conf.IdleTimeout == 0 {
		conf.IdleTimeout = 60 * time.Second
	}

	addr := fmt.Sprintf(":%d", conf.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  conf.ReadTimeout,
		WriteTimeout: conf.WriteTimeout,
		IdleTimeout:  conf.IdleTimeout,
		ErrorLog:     zap.NewStdLog(logger.Named("http-server")),
	}

	return &HTTPServer{
		conf:   conf,
		server: server,
		logger: logger.Named("http"),
	}
}

func (h *HTTPServer) Run(ctx context.Context) error {
	h.logger.Info("HTTP server starting", zap.String("address", h.server.Addr))

	errChan := make(chan error, 1)

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP server ListenAndServe failed", zap.Error(err))
			errChan <- err
			return
		}
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		h.logger.Info("HTTP server shutting down due to context cancellation")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := h.server.Shutdown(shutdownCtx); err != nil {
			h.logger.Error("HTTP server shutdown failed", zap.Error(err))
			return err
		}
		h.logger.Info("HTTP server stopped successfully")
		return nil

	case err := <-errChan: // ListenAndServe exited with an error
		// If errChan is closed without a value, err will be nil.
		// This happens if ListenAndServe returned http.ErrServerClosed or nil.
		// In such cases, it means the server was likely shut down gracefully
		if err != nil {
			h.logger.Error("HTTP server failed to start or encountered a runtime error", zap.Error(err))
			return err
		}

		h.logger.Info("HTTP server ListenAndServe finished")
		return nil
	}
}
