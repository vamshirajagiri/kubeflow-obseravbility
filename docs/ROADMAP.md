# Roadmap

## Phase 1

Research Kubeflow Pipelines architecture.

Status:

In Progress

Current focus:

- understand KFP run data sources
- compare API, database, and Kubernetes/Argo sources
- document the run data model
- decide the first exporter source of truth

---

## Phase 2

Identify available APIs.

Status:

In Progress

Current focus:

- v2beta1 runs API
- experiments API
- recurring runs API
- auth behavior
- pagination and filtering

---

## Phase 3

Design exporter architecture.

Status:

In Progress

Current focus:

- define metric contract
- choose safe labels
- avoid historical counters until deduplication is designed
- define first polling behavior

---

## Phase 4

Implement collectors.

Status:

In Progress

Current focus:

- exporter skeleton
- configuration parsing
- health endpoint
- metrics endpoint
- exporter self-metrics

---

## Phase 5

Expose Prometheus metrics.

Status:

Not Started

---

## Phase 6

Testing.

Status:

Not Started

---

## Phase 7

Helm Chart.

Status:

Not Started

---

## Phase 8

Grafana Dashboard.

Status:

Not Started

---

## Phase 9

Prepare upstream proposal.

Status:

Not Started
