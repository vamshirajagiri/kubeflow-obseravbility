# Research Notes

This file captures what we learn before writing production code.

The goal is not to collect random facts. The goal is to understand which data
source can support a small, reliable Prometheus exporter.

## First Principles

An exporter is a translator.

It reads facts from one system and exposes those facts in Prometheus format.
If the source facts are wrong, expensive to fetch, unstable across versions, or
too detailed, the exporter becomes noisy and hard to operate.

For this project, the first production question is:

Which Kubeflow Pipelines data source should be treated as the source of truth
for pipeline run metrics?

## Current Upstream Context

Kubeflow Pipelines is a service for reusable machine learning workflows on
Kubernetes. The upstream repository says KFP uses Argo Workflows by default for
orchestration, and the current compatibility notes mention Argo Workflows v3.7
and v4.0 plus MySQL v8.

Sources:

- https://github.com/kubeflow/pipelines
- https://www.kubeflow.org/docs/components/pipelines/

As of the first research pass, GitHub shows Kubeflow Pipelines release 2.17.0
as the latest release, dated 2026-07-09. The Kubeflow API reference page still
links its generated Swagger UI to KFP 2.16.1, so we should treat the docs and
latest source as close but not identical.

## Candidate Data Sources

### 1. Kubeflow Pipelines REST API

The v2beta1 REST API exposes pipeline runs under:

```text
GET /apis/v2beta1/runs
GET /apis/v2beta1/runs/{run_id}
```

The API supports namespace filtering:

```text
GET /pipeline/apis/v2beta1/runs?namespace=team-1
```

The `ListRunsRequest` supports:

- `namespace`
- `experiment_id`
- `page_token`
- `page_size`
- `sort_by`
- `filter`

The `ListRunsResponse` returns:

- `runs`
- `total_size`
- `next_page_token`

This is a strong exporter source because pagination and namespace filtering are
already part of the public API contract.

Sources:

- https://www.kubeflow.org/docs/components/pipelines/reference/api/kubeflow-pipeline-api-spec/
- https://github.com/kubeflow/pipelines/blob/master/backend/api/v2beta1/run.proto

### 2. Kubeflow Pipelines Database

The backend model stores runs in a table named `run_details`.

Important fields include:

- `UUID`
- `DisplayName`
- `Name`
- `Namespace`
- `ExperimentUUID`
- `JobUUID`
- `StorageState`
- `ServiceAccount`
- `CreatedAtInSec`
- `ScheduledAtInSec`
- `FinishedAtInSec`
- `Conditions`
- `State`
- `StateHistory`
- `PipelineRuntimeManifest`
- `WorkflowRuntimeManifest`
- `PipelineContextId`
- `PipelineRunContextId`

This is rich data, but it is not the best first exporter source.

Why:

- It couples us to KFP internals.
- It requires database credentials.
- It may change across KFP versions.
- It bypasses KFP authorization and namespace isolation.
- It makes deployment heavier for platform teams.

Real-world example:

A Prometheus exporter that needs read-only HTTP access to a service is easy to
deploy. An exporter that needs database credentials needs secret management,
network policy exceptions, database load review, and schema compatibility
testing. That is a lot of operational weight before the product has proven its
core value.

Source:

- https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/model/run.go

### 3. Kubernetes / Argo Workflows

KFP uses Argo Workflows by default under the hood. That means Kubernetes
resources can tell us about workflow execution.

This is useful for runtime details, but it is not the cleanest first source for
pipeline-level metrics.

Why:

- The KFP API knows KFP concepts such as experiments, recurring runs, pipeline
  versions, and storage state.
- Argo knows workflow execution, but not always the higher-level product model
  that users see in the KFP UI.
- Some deployments may move toward Kubernetes native API mode, which changes
  which resources matter.

Source:

- https://github.com/kubeflow/pipelines
- https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/

### 4. Existing Metrics

This pass did not confirm a stable, complete, built-in metric set that answers
the project questions:

- Which pipelines are currently running?
- Which pipelines fail most often?
- How long do runs take?
- How many runs are active per namespace?
- Which recurring runs are enabled or disabled?

This remains an open research item. We should inspect a running KFP deployment
or upstream manifests before claiming there are no useful existing metrics.

## Run Lifecycle

KFP v2 runtime states are:

- `PENDING`
- `RUNNING`
- `SUCCEEDED`
- `SKIPPED`
- `FAILED`
- `CANCELING`
- `CANCELED`
- `PAUSED`

There is also `RUNTIME_STATE_UNSPECIFIED`, but we should treat that as unknown
or invalid for metrics instead of as a meaningful user-facing state.

Important timestamps:

- `created_at`: when the run record is created
- `scheduled_at`: when the run was scheduled to start
- `finished_at`: when the run completed

The API also exposes `state_history`, which records state transitions.

Source:

- https://github.com/kubeflow/pipelines/blob/master/backend/api/v2beta1/run.proto

## Experiments, Namespaces, and Isolation

An experiment is a workspace for grouping pipeline runs.

In multi-user Kubeflow Pipelines deployments:

- experiments belong to namespaces
- runs and recurring runs belong to their parent experiment namespace
- pipeline runs execute in user namespaces
- the KFP API server rejects namespace requests the caller is not authorized to
  access

This matters for exporter design.

The exporter should not list Kubernetes namespaces directly in the default
design. It should query the KFP API and collect the KFP runs visible to the
exporter identity.

That means the default scope is:

```text
all KFP-visible runs
```

not:

```text
all Kubernetes namespaces
```

This distinction matters.

Kubeflow may run in a cluster that also has namespaces such as `kube-system`,
`cert-manager`, or `observability`. Those namespaces are not automatically KFP
pipeline namespaces. The KFP API is the product boundary for this exporter.

Namespace filtering should be optional. It is useful when a platform team wants
to monitor only a subset of KFP namespaces, but requiring an allow-list by
default creates unnecessary operational work for platform-wide monitoring.

Sources:

- https://www.kubeflow.org/docs/components/pipelines/concepts/experiment/
- https://www.kubeflow.org/docs/components/pipelines/operator-guides/multi-user/

## Recurring Runs

Recurring runs are stored internally as jobs. The backend model has a `jobs`
table-like model with fields such as:

- `UUID`
- `DisplayName`
- `Namespace`
- `ServiceAccount`
- `MaxConcurrency`
- `NoCatchup`
- `CreatedAtInSec`
- `UpdatedAtInSec`
- `Enabled`
- `ExperimentUUID`
- `Conditions`

This is likely the source for scheduled pipeline metrics, but we still need to
confirm the v2beta1 public API endpoint and response shape before designing
metrics around it.

Source:

- https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/model/job.go

## Initial Finding

For the first production-grade version, prefer the Kubeflow Pipelines REST API
over direct database access.

Reason:

The API is the stable product contract. The database is an implementation
detail.

This does not mean the API is perfect. It means it is the right first boundary.
Production systems usually become easier to maintain when they depend on public
contracts instead of internal tables.

## Open Questions

- Which KFP versions should the exporter support first: v2 only, or v1 and v2?
- What auth method should be the default inside a Kubeflow cluster?
- What page size is safe for listing runs without putting pressure on the KFP
  API server?
- Can the API filter by state and time range efficiently enough for scraping?
- Are recurring runs available through a stable v2beta1 endpoint in all target
  versions?
- Does the KFP v2 list-runs API return all KFP-visible namespaces by default in
  every supported deployment mode?
- Which existing KFP endpoints already expose Prometheus metrics?
- Should run display name or pipeline display name be used as a metric label, or
  should we avoid them to control cardinality?
- How should we handle archived runs?
- How should we prevent old completed runs from being re-counted incorrectly
  across exporter restarts?
