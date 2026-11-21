package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kubeflow-pipelines-exporter/internal/config"
	"kubeflow-pipelines-exporter/internal/kfp"
	"kubeflow-pipelines-exporter/internal/poller"
	"kubeflow-pipelines-exporter/internal/server"
	"kubeflow-pipelines-exporter/internal/state"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.FromArgs(os.Args[1:], os.LookupEnv)
	if err != nil {
		logger.Error("parse config", "err", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := state.NewStore(time.Now())
	kfpClient, err := kfp.NewClient(
		cfg.KFPAPIURL,
		cfg.PageSize,
		cfg.RequestTimeout,
		kfp.WithBearerTokenFile(cfg.BearerTokenFile),
	)
	if err != nil {
		logger.Error("create kfp client", "err", err)
		os.Exit(2)
	}

	p := poller.New(cfg, kfpClient, store, logger)
	go p.Run(ctx)

	srv := server.New(cfg, store, logger)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting exporter", "addr", cfg.ListenAddress)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server", "err", err)
			os.Exit(1)
		}
		logger.Info("exporter stopped")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("run server", "err", err)
			os.Exit(1)
		}
	}
}
