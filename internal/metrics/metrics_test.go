package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"kubeflow-pipelines-exporter/internal/state"
)

func TestWriteText(t *testing.T) {
	var buf bytes.Buffer
	store := state.NewStore(time.Now().Add(-time.Second))
	store.RecordPollSuccess(map[string]map[string]int{
		"team-a": {"RUNNING": 2},
	}, 2, 100*time.Millisecond, 1)

	if err := WriteText(&buf, store.Snapshot()); err != nil {
		t.Fatalf("WriteText returned error: %v", err)
	}

	text := buf.String()
	for _, want := range []string{
		"# HELP kubeflow_pipelines_exporter_build_info",
		"# TYPE kubeflow_pipelines_exporter_build_info gauge",
		"kubeflow_pipelines_exporter_build_info",
		"# HELP kubeflow_pipelines_exporter_uptime_seconds",
		"# TYPE kubeflow_pipelines_exporter_uptime_seconds gauge",
		"kubeflow_pipelines_exporter_uptime_seconds",
		"kubeflow_pipelines_exporter_poll_success 1",
		`kubeflow_pipelines_exporter_api_requests_total{source="runs",result="success"} 1`,
		`kubeflow_pipelines_runs_current{namespace="team-a",state="RUNNING"} 2`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, text)
		}
	}
}
