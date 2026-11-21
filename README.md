# Kubeflow Pipelines Exporter

A Prometheus exporter for Kubeflow Pipelines v2.

The project is in early development. The current binary exposes exporter
self-metrics only. KFP API polling will be added after the polling and metric
contract are finalized.

## Run Locally

```sh
go run ./cmd/exporter
```

Then open:

```text
http://localhost:8080/healthz
http://localhost:8080/metrics
```

## Configuration

Flags:

```text
-listen-address string
    address for the HTTP server (default ":8080")
-kfp-api-url string
    Kubeflow Pipelines API base URL
-namespaces string
    comma-separated KFP namespaces to collect; empty means all KFP-visible
    namespaces
-poll-interval duration
    interval between KFP API polls (default 1m)
```

Environment variables:

```text
KFP_EXPORTER_LISTEN_ADDRESS
KFP_EXPORTER_KFP_API_URL
KFP_EXPORTER_NAMESPACES
KFP_EXPORTER_POLL_INTERVAL
```

Flags override environment variables.

## Namespace Behavior

By default, the exporter should collect all Kubeflow Pipelines data visible to
the exporter identity through the KFP API.

This does not mean it lists every Kubernetes namespace. It means it asks KFP for
the KFP runs the exporter is authorized to see.

Use `-namespaces` when you want to restrict collection:

```sh
go run ./cmd/exporter -namespaces=team-a,team-b
```

## Current Metrics

```text
kubeflow_pipelines_exporter_build_info
kubeflow_pipelines_exporter_uptime_seconds
```

KFP run metrics are documented in `docs/METRICS.md` but are not implemented
yet.
