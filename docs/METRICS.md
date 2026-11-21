# Metrics Contract

This document defines the Prometheus metrics this exporter intends to expose.

The first target is Kubeflow Pipelines v2.

## Why This Document Exists

Metrics are a product API.

Once users build dashboards and alerts on metric names, labels, and meanings,
changing them becomes painful. That is why we design the metric contract before
writing collectors.

Real-world example:

If a team creates an alert on:

```promql
kubeflow_pipelines_runs_current{state="FAILED"} > 10
```

and we later rename the metric or change what `state` means, their alert breaks
or becomes misleading. Stable metrics are part of production quality.

## First Principles

Prometheus stores time series.

A time series is created by:

- metric name
- full label set

Example:

```text
kubeflow_pipelines_runs_current{namespace="team-a",state="RUNNING"}
```

This is one time series.

If `namespace` has 20 values and `state` has 8 values, this metric can create up
to 160 time series. That is usually fine.

If we add `run_id`, every pipeline run creates a new time series. That grows
forever and becomes expensive. This is called high cardinality.

First rule:

Do not use unbounded values as labels in default metrics.

Unbounded values include:

- run ID
- run display name
- workflow ID
- pod name
- user email
- arbitrary pipeline parameter values

## Metric Naming Rules

All project metrics use this prefix:

```text
kubeflow_pipelines_
```

Why plural `pipelines`?

The product being observed is Kubeflow Pipelines, not one pipeline object.

Rules:

- use lowercase names
- use underscores
- include the unit suffix when there is a unit
- use `_total` only for counters
- use `_seconds` for durations
- avoid names that include implementation details such as `argo` or `mysql`
  unless the metric truly describes that implementation

## Metric Types

### Gauge

A gauge is a value that can go up and down.

Real-world examples:

- current running pipeline runs
- last successful poll timestamp
- number of API errors seen during the last poll

Use a gauge for "what is true right now?"

### Counter

A counter only goes up, until the process restarts.

Real-world examples:

- total exporter poll attempts
- total exporter poll failures

Use a counter for "how many times has this happened since this process
started?"

Important:

Counters are not good for historical KFP runs unless the exporter has a safe
way to avoid double-counting after restart. That is why completed-run counters
are not in the first metric set.

### Histogram

A histogram observes a distribution.

Real-world examples:

- pipeline run duration
- KFP API request duration

Histograms are powerful, but they create multiple time series per label set. Use
them only when the product value is clear.

## Label Policy

Allowed default labels for v0:

- `namespace`
- `state`
- `source`

Possible later labels:

- `pipeline_id`
- `pipeline_version_id`
- `experiment_id`

Risky labels, rejected for v0:

- `run_id`
- `run_name`
- `display_name`
- `workflow_name`
- `pod_name`
- `user`
- `service_account`

Why:

Production monitoring systems need stable cardinality. Labels like namespace and
state have controlled value sets. Labels like run ID and display name grow with
workload volume.

## v0 Metric Set

The first metric set should answer a small number of operational questions:

- Is the exporter healthy?
- Can it reach Kubeflow Pipelines?
- How many runs are currently in each state?
- Is the data fresh?

This is intentionally small. A small correct metric set is better than a large
confusing one.

Implemented in the first skeleton:

- `kubeflow_pipelines_exporter_build_info`
- `kubeflow_pipelines_exporter_uptime_seconds`

These prove the `/metrics` endpoint works before KFP polling exists.

### kubeflow_pipelines_exporter_build_info

Type: Gauge

Description:

Build and runtime information for the exporter process.

Labels:

- `go_version`

Value:

- always `1`

Source:

- exporter runtime

Why this matters:

This gives operators a simple way to see that the exporter is alive and which
Go runtime built the current process.

### kubeflow_pipelines_exporter_uptime_seconds

Type: Gauge

Description:

Seconds since the exporter process started.

Labels:

- none

Source:

- exporter runtime

Why this matters:

This helps identify restarts before we add Kubernetes deployment metrics.

### kubeflow_pipelines_runs_current

Type: Gauge

Description:

Current number of Kubeflow Pipelines v2 runs grouped by namespace and runtime
state.

Labels:

- `namespace`
- `state`

Source:

- `GET /apis/v2beta1/runs`
- KFP run field: `state`
- default: all KFP-visible runs returned by the API
- optional: KFP namespace filter when configured

Example:

```text
kubeflow_pipelines_runs_current{namespace="team-a",state="RUNNING"} 3
kubeflow_pipelines_runs_current{namespace="team-a",state="FAILED"} 1
```

Why this matters:

Platform teams need to know whether pipeline execution is stuck, failing, or
backing up.

Open questions:

- Should terminal states include all historical runs, or only a bounded recent
  window?
- Can the API filter by state efficiently?
- Can the API filter by updated time or created time?

v0 rule:

Only expose this metric after we define the polling window. Reporting every
historical completed run as "current" would be misleading.

### kubeflow_pipelines_exporter_last_poll_timestamp_seconds

Type: Gauge

Description:

Unix timestamp of the last successful Kubeflow Pipelines poll.

Labels:

- none

Source:

- exporter internal state

Example:

```text
kubeflow_pipelines_exporter_last_poll_timestamp_seconds 1783756200
```

Why this matters:

Prometheus may still scrape the exporter even when the exporter has not
successfully talked to KFP for several minutes. This metric shows data freshness.

### kubeflow_pipelines_exporter_poll_success

Type: Gauge

Description:

Whether the last Kubeflow Pipelines poll succeeded.

Labels:

- none

Values:

- `1` means success
- `0` means failure

Source:

- exporter internal state

Example:

```text
kubeflow_pipelines_exporter_poll_success 1
```

Why this matters:

This is the simple "can I currently read KFP?" signal.

### kubeflow_pipelines_exporter_poll_duration_seconds

Type: Histogram

Description:

Duration of Kubeflow Pipelines polling attempts.

Labels:

- `source`

Allowed `source` values:

- `runs`
- later: `experiments`
- later: `recurring_runs`

Source:

- exporter internal measurement

Why this matters:

If KFP API calls become slow, users should see that before exporter data becomes
stale.

v0 note:

This metric is about exporter behavior, not pipeline run duration.

### kubeflow_pipelines_exporter_api_requests_total

Type: Counter

Description:

Total number of KFP API requests made by the exporter.

Labels:

- `source`
- `result`

Allowed `source` values:

- `runs`
- later: `experiments`
- later: `recurring_runs`

Allowed `result` values:

- `success`
- `error`

Source:

- exporter internal HTTP client instrumentation

Example:

```text
kubeflow_pipelines_exporter_api_requests_total{source="runs",result="success"} 42
kubeflow_pipelines_exporter_api_requests_total{source="runs",result="error"} 2
```

Why this matters:

This helps answer whether the exporter is putting pressure on KFP or repeatedly
failing.

## Later Metrics

These are useful, but should wait until the data source behavior is better
understood.

### kubeflow_pipelines_run_duration_seconds

Type: Histogram

Description:

Duration of completed pipeline runs.

Possible labels:

- `namespace`
- maybe `state`

Rejected labels:

- `run_id`
- `run_name`
- `display_name`

Why later:

Completed-run duration needs careful deduplication. If the exporter restarts and
re-reads old completed runs, it can observe the same run duration again and
corrupt the histogram.

### kubeflow_pipelines_recurring_runs_current

Type: Gauge

Description:

Current number of recurring runs grouped by namespace and enabled state.

Possible labels:

- `namespace`
- `enabled`

Why later:

We need to confirm the stable KFP v2 public API shape for recurring runs.

### kubeflow_pipelines_experiments_current

Type: Gauge

Description:

Current number of experiments grouped by namespace.

Possible labels:

- `namespace`

Why later:

Useful, but less urgent than run visibility.

## Rejected v0 Metrics

### kubeflow_pipelines_runs_total

Reason:

This sounds useful, but total historical runs are hard to expose correctly as a
counter unless the exporter owns event ingestion or durable state.

If the KFP API returns all historical runs and the exporter turns them into a
counter, restarts can cause double-counting.

Better first metric:

```text
kubeflow_pipelines_runs_current
```

### kubeflow_pipelines_failed_runs_total

Reason:

Same counter problem as `kubeflow_pipelines_runs_total`.

Better first approach:

Expose current or recent-window failed runs as a gauge after we define the
window.

### kubeflow_pipelines_run_info

Reason:

Info metrics with `run_id` or `display_name` labels create high cardinality.

They may be useful for debugging in an optional mode later, but should not be
enabled by default.

## v0 Open Questions

- What is the default polling interval?
- What is the default API page size?
- Should `kubeflow_pipelines_runs_current` include terminal states?
- If terminal states are included, what time window defines "current"?
- Which auth mode should be documented first for in-cluster deployment?
- Should API request metrics include HTTP status code, or is `result` enough for
  v0?
- Does the KFP v2 runs API return all KFP-visible namespaces by default in every
  deployment mode we support?
