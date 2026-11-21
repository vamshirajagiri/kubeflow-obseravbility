# Current Task

## Goal

Build the first exporter skeleton without KFP polling.

## Success Criteria

- Go module exists.
- Binary starts an HTTP server.
- `/healthz` returns success.
- `/metrics` returns Prometheus text format.
- Configuration supports flags and environment variables.
- Namespace configuration defaults to all KFP-visible namespaces.
- Tests pass.

## Deliverables

- `cmd/exporter/main.go`
- `internal/config`
- `internal/metrics`
- `internal/server`
- README update
- Documentation update

Status

In Progress.

Deployment direction:

Deploy one exporter near the Kubeflow control plane. By default, collect all
KFP-visible runs returned by the KFP API. Allow optional namespace filters for
teams that need restricted monitoring.

Current implementation boundary:

The exporter skeleton does not call KFP yet. It only proves that the process,
configuration, health endpoint, and metrics endpoint work.
