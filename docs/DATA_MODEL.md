# Data Model

This document explains the Kubeflow Pipelines objects that matter for the first
exporter design.

## Why This Exists

Before building metrics, we need to know what real-world thing each metric is
describing.

Example:

`pipeline_runs_total` sounds simple, but what is a "run"?

- a KFP run record?
- an Argo workflow?
- a pipeline version execution?
- a retry attempt?
- a recurring run occurrence?

If we do not define the data model, we will create metrics that look useful but
are confusing in production.

## Core Objects

### Pipeline

A reusable workflow definition.

Real-world example:

"Train fraud detection model" can be a pipeline.

### Pipeline Version

A specific version of a pipeline definition.

Real-world example:

Version 1 trains with XGBoost. Version 2 trains with LightGBM. They are the same
business pipeline idea, but different executable definitions.

### Experiment

A workspace that groups runs.

Real-world example:

"Fraud model tuning - July" can contain many runs with different parameters.

In multi-user Kubeflow, experiments belong to namespaces.

### Run

One execution of a pipeline.

Real-world example:

"Run the fraud training pipeline today with learning_rate=0.05."

Important API fields:

- `run_id`
- `display_name`
- `experiment_id`
- `storage_state`
- `pipeline_version_reference`
- `runtime_config`
- `service_account`
- `created_at`
- `scheduled_at`
- `finished_at`
- `state`
- `state_history`
- `recurring_run_id`
- `run_details`

### Recurring Run

A schedule that creates runs repeatedly.

Real-world example:

"Run the fraud training pipeline every day at 1 AM."

The backend model stores recurring-run-like data as jobs with fields such as
`Enabled`, `MaxConcurrency`, `NoCatchup`, and `ExperimentUUID`.

## Run States

Known KFP v2 run states:

- `PENDING`
- `RUNNING`
- `SUCCEEDED`
- `SKIPPED`
- `FAILED`
- `CANCELING`
- `CANCELED`
- `PAUSED`

Exporter guidance:

- active states: `PENDING`, `RUNNING`, `CANCELING`, `PAUSED`
- successful terminal states: `SUCCEEDED`, `SKIPPED`
- failed terminal states: `FAILED`, `CANCELED`
- unknown state: `RUNTIME_STATE_UNSPECIFIED`

This grouping is only an initial proposal. It must be validated against real
KFP behavior before metric names become stable.

## Important Timestamps

### Created At

When KFP created the run record.

### Scheduled At

When the run was supposed to start.

This can differ from `created_at`, especially for backfilled scheduled runs.

### Finished At

When execution completed.

This enables duration metrics:

```text
duration = finished_at - scheduled_at
```

or sometimes:

```text
duration = finished_at - created_at
```

We must choose and document one definition before exposing a stable metric.

## Metrics Design Implication

A good metric should map to one clear data model concept.

Good:

```text
kubeflow_pipelines_runs_current{namespace="team-a",state="RUNNING"}
```

This means current run records in a state.

Risky:

```text
kubeflow_pipelines_run_info{run_id="...",display_name="..."}
```

This can create huge label cardinality because every run creates a new time
series.

First principle:

Prometheus labels are indexes. Indexes with unbounded values become expensive.
Run IDs are unbounded, so they should not be labels in the default metrics.
