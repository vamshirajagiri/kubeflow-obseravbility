# Exporter Architecture

This document describes the intended shape of the exporter. It is still a
design document, not an implementation contract.

## Product Philosophy

The exporter should be boring in the best way.

That means:

- one small binary
- clear configuration
- low memory usage
- low API pressure on Kubeflow
- predictable Prometheus metrics
- no direct database dependency for the first version
- no clever abstractions before they pay rent

This follows the same broad product taste that makes systems like
VictoriaMetrics attractive: solve the real operational problem, keep the runtime
simple, and make the product understandable by reading the code and docs.

## First Principles

Prometheus does not push work into exporters.

Prometheus scrapes an HTTP endpoint, usually `/metrics`. When a scrape happens,
the exporter must either:

1. return already cached metric values, or
2. fetch fresh data from the upstream system and translate it.

For Kubeflow Pipelines, fetching everything on every scrape may become expensive
in large clusters. So the architecture should separate:

- polling Kubeflow
- storing the latest observed snapshot in memory
- exposing that snapshot to Prometheus

That keeps scrape latency predictable.

Real-world example:

If Prometheus scrapes every 15 seconds and the exporter calls the KFP API on
every scrape, three Prometheus replicas could triple API traffic. If the exporter
polls KFP every 60 seconds and serves cached results, Prometheus scrape behavior
does not directly multiply Kubeflow API load.

## Initial Data Source Decision

Use the Kubeflow Pipelines REST API as the first data source.

Do not read the KFP database directly in the first version.

Why:

- the API is the public product boundary
- the API understands auth and namespace isolation
- deployment is simpler
- schema changes are less likely to break us

Tradeoff:

- some detailed fields may be unavailable or slower to query through the API
- we may need careful pagination and filtering

## High-Level Flow

```text
Kubeflow Pipelines API
        |
        v
KFP API client
        |
        v
Polling loop
        |
        v
In-memory snapshot
        |
        v
Prometheus collectors
        |
        v
HTTP /metrics endpoint
        |
        v
Prometheus
```

## Components

### Configuration

Responsible for reading flags and environment variables.

Current configuration:

- `listen-address`
- `kfp-api-url`
- `namespaces`
- `poll-interval`

First principle:

Configuration should describe product behavior, not implementation accidents.
For example, an empty namespace list means "all KFP-visible namespaces" because
that is the platform default we want. It does not mean "forgot to configure the
exporter."

### API Client

Responsible for talking to Kubeflow Pipelines.

It should hide details like:

- base URL
- auth headers
- pagination
- request timeout
- optional namespace query parameter

It should not know Prometheus concepts.

### Poller

Responsible for periodically asking the API client for current KFP state.

It should:

- run on a configurable interval
- use context timeouts
- record scrape source errors
- keep the last successful snapshot if a poll fails

This is the same reliability idea used in many monitoring agents: temporary
upstream failure should be visible, but it should not make the exporter itself
crash.

### Snapshot

An in-memory representation of the latest observed runs, experiments, and later
recurring runs.

The snapshot should contain only what metrics need.

This keeps the design honest. If a field does not feed a metric, alert, or
debugging signal, it should not be stored yet.

### Prometheus Collectors

Responsible for converting the snapshot into Prometheus metrics.

Collectors should not call Kubeflow directly. They should read from the latest
snapshot.

### HTTP Server

Responsible for:

- `/metrics`
- `/healthz`
- maybe `/readyz`

Keep it simple. No web framework is needed at this stage.

The first implementation uses Go's standard `net/http` server.

Why:

The exporter only needs a few endpoints. A web framework would add dependency
weight before solving a real problem.

## First Deployment Shape

Start with one exporter instance deployed near the Kubeflow control plane,
usually in the namespace where Kubeflow Pipelines is deployed.

Default behavior:

```text
one exporter -> KFP API -> all KFP-visible runs
```

This does not mean the exporter lists every Kubernetes namespace. The exporter
is not a generic Kubernetes namespace crawler. It asks the Kubeflow Pipelines
API for pipeline objects the exporter identity is authorized to see.

Optional restricted behavior:

```text
one exporter -> KFP API -> configured namespaces only
```

This mode is useful when a platform team wants to expose metrics for only some
KFP namespaces.

Per-namespace exporter deployments are not the default product path. They can be
supported later for strict tenant isolation, but they add repeated pods,
scrape targets, and configuration.

First principle:

Authorization says what the exporter identity can read. Exporter configuration
says what the exporter chooses to collect and expose.

For v0, the product should optimize for the normal platform-engineering use
case: deploy once, observe Kubeflow Pipelines as a platform service, and allow
namespace restriction when needed.

## Non-Goals For The First Version

- direct database reads
- custom Kubernetes controllers
- Grafana dashboard generation
- alert rule generation
- long-term storage
- tracing
- per-run high-cardinality labels
- support for every KFP version on day one

## Technical Unknowns

- exact auth method for in-cluster exporter deployments
- efficient filtering by run state and time range
- recurring run API shape for the target KFP version
- whether KFP has built-in Prometheus metrics worth reusing
- correct handling of archived runs
- safe metric labels for pipeline identity
- whether v1 compatibility is required
- whether the KFP list-runs API returns all visible namespaces by default in
  every supported v2 deployment mode
