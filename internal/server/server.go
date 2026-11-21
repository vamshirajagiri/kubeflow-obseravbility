package server

import (
	"log/slog"
	"net/http"
	"time"

	"kubeflow-pipelines-exporter/internal/config"
	"kubeflow-pipelines-exporter/internal/metrics"
	"kubeflow-pipelines-exporter/internal/state"
)

func New(cfg config.Config, store *state.Store, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc(cfg.HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc(cfg.MetricsPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := metrics.WriteText(w, store.Snapshot()); err != nil {
			logger.Error("write metrics", "err", err)
		}
	})

	return &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
