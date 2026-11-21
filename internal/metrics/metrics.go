package metrics

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"time"

	"kubeflow-pipelines-exporter/internal/state"
)

func WriteText(w io.Writer, snapshot state.Snapshot) error {
	uptime := time.Since(snapshot.StartTime).Seconds()
	if uptime < 0 {
		uptime = 0
	}
	pollSuccess := 0
	if snapshot.LastPollSuccess {
		pollSuccess = 1
	}

	lines := []string{
		"# HELP kubeflow_pipelines_exporter_build_info Build information for the Kubeflow Pipelines exporter.",
		"# TYPE kubeflow_pipelines_exporter_build_info gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_build_info{go_version=%q} 1", runtime.Version()),
		"# HELP kubeflow_pipelines_exporter_uptime_seconds Seconds since the exporter process started.",
		"# TYPE kubeflow_pipelines_exporter_uptime_seconds gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_uptime_seconds %.3f", uptime),
		"# HELP kubeflow_pipelines_exporter_last_poll_timestamp_seconds Unix timestamp of the last KFP poll attempt.",
		"# TYPE kubeflow_pipelines_exporter_last_poll_timestamp_seconds gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_last_poll_timestamp_seconds %.0f", timestampSeconds(snapshot.LastPollTime)),
		"# HELP kubeflow_pipelines_exporter_poll_success Whether the last KFP poll succeeded. 1 means success, 0 means failure.",
		"# TYPE kubeflow_pipelines_exporter_poll_success gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_poll_success %d", pollSuccess),
		"# HELP kubeflow_pipelines_exporter_poll_duration_seconds Duration of the last KFP poll attempt.",
		"# TYPE kubeflow_pipelines_exporter_poll_duration_seconds gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_poll_duration_seconds %.3f", snapshot.LastPollDuration.Seconds()),
		"# HELP kubeflow_pipelines_exporter_last_poll_runs Number of runs returned by the last successful KFP poll before active-state filtering.",
		"# TYPE kubeflow_pipelines_exporter_last_poll_runs gauge",
		fmt.Sprintf("kubeflow_pipelines_exporter_last_poll_runs %d", snapshot.LastPollRunCount),
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	if err := writeAPIRequests(w, snapshot.APIRequests); err != nil {
		return err
	}
	if err := writeRunsCurrent(w, snapshot.RunsCurrent); err != nil {
		return err
	}

	return nil
}

func writeAPIRequests(w io.Writer, requests map[string]map[string]uint64) error {
	if _, err := fmt.Fprintln(w, "# HELP kubeflow_pipelines_exporter_api_requests_total Total KFP API requests made by the exporter."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE kubeflow_pipelines_exporter_api_requests_total counter"); err != nil {
		return err
	}

	for _, source := range sortedMapKeys(requests) {
		for _, result := range sortedMapKeys(requests[source]) {
			if _, err := fmt.Fprintf(w, "kubeflow_pipelines_exporter_api_requests_total{source=%q,result=%q} %d\n", source, result, requests[source][result]); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeRunsCurrent(w io.Writer, runs map[string]map[string]int) error {
	if _, err := fmt.Fprintln(w, "# HELP kubeflow_pipelines_runs_current Current active Kubeflow Pipelines v2 runs grouped by namespace and runtime state."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE kubeflow_pipelines_runs_current gauge"); err != nil {
		return err
	}

	for _, namespace := range sortedMapKeys(runs) {
		for _, runState := range sortedMapKeys(runs[namespace]) {
			if _, err := fmt.Fprintf(w, "kubeflow_pipelines_runs_current{namespace=%q,state=%q} %d\n", namespace, runState, runs[namespace][runState]); err != nil {
				return err
			}
		}
	}
	return nil
}

func timestampSeconds(value time.Time) float64 {
	if value.IsZero() {
		return 0
	}
	return float64(value.Unix())
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
