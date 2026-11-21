package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kubeflow-pipelines-exporter/internal/config"
	"kubeflow-pipelines-exporter/internal/state"
)

func TestHealthHandler(t *testing.T) {
	srv := New(testConfig(), state.NewStore(time.Now()), slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok", rec.Body.String())
	}
}

func TestMetricsHandler(t *testing.T) {
	store := state.NewStore(time.Now())
	store.RecordPollSuccess(map[string]map[string]int{
		"team-a": {"RUNNING": 1},
	}, 1, time.Millisecond, 1)
	srv := New(testConfig(), store, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	if !strings.Contains(rec.Body.String(), "kubeflow_pipelines_exporter_build_info") {
		t.Fatalf("metrics body missing build info:\n%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `kubeflow_pipelines_runs_current{namespace="team-a",state="RUNNING"} 1`) {
		t.Fatalf("metrics body missing run count:\n%s", rec.Body.String())
	}
}

func testConfig() config.Config {
	return config.Config{
		ListenAddress:   ":0",
		MetricsPath:     "/metrics",
		HealthPath:      "/healthz",
		KFPAPIURL:       "http://kfp.example",
		PollInterval:    time.Minute,
		ShutdownTimeout: time.Second,
	}
}
