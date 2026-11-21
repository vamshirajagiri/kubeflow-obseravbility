package poller

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"kubeflow-pipelines-exporter/internal/config"
	"kubeflow-pipelines-exporter/internal/kfp"
	"kubeflow-pipelines-exporter/internal/state"
)

func TestPollerRecordsActiveRuns(t *testing.T) {
	store := state.NewStore(time.Now())
	client := &fakeClient{
		runs: []kfp.Run{
			{ID: "1", Namespace: "team-a", State: "RUNNING"},
			{ID: "2", Namespace: "team-a", State: "SUCCEEDED"},
			{ID: "3", Namespace: "team-b", State: "PENDING"},
		},
	}

	p := New(testConfig(), client, store, slog.Default())
	p.poll(context.Background())

	snapshot := store.Snapshot()
	if !snapshot.LastPollSuccess {
		t.Fatalf("LastPollSuccess = false")
	}
	if got := snapshot.RunsCurrent["team-a"]["RUNNING"]; got != 1 {
		t.Fatalf("team-a RUNNING = %d, want 1", got)
	}
	if got := snapshot.RunsCurrent["team-a"]["SUCCEEDED"]; got != 0 {
		t.Fatalf("team-a SUCCEEDED = %d, want 0", got)
	}
	if got := snapshot.RunsCurrent["team-b"]["PENDING"]; got != 1 {
		t.Fatalf("team-b PENDING = %d, want 1", got)
	}
}

func TestPollerUsesConfiguredNamespaces(t *testing.T) {
	store := state.NewStore(time.Now())
	client := &fakeClient{}

	cfg := testConfig()
	cfg.Namespaces = []string{"team-a", "team-b"}

	p := New(cfg, client, store, slog.Default())
	p.poll(context.Background())

	if got := len(client.namespaces); got != 2 {
		t.Fatalf("namespace calls = %d, want 2", got)
	}
	if client.namespaces[0] != "team-a" || client.namespaces[1] != "team-b" {
		t.Fatalf("namespaces = %#v", client.namespaces)
	}
}

type fakeClient struct {
	runs       []kfp.Run
	namespaces []string
}

func (f *fakeClient) ListRuns(_ context.Context, namespace string) ([]kfp.Run, int, error) {
	f.namespaces = append(f.namespaces, namespace)
	return f.runs, 1, nil
}

func testConfig() config.Config {
	return config.Config{
		KFPAPIURL:       "http://kfp.example",
		PollInterval:    time.Minute,
		PageSize:        100,
		RequestTimeout:  time.Second,
		ShutdownTimeout: time.Second,
	}
}
