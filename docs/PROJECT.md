# Project Vision

## Goal

Build an enterprise-grade Prometheus exporter for Kubeflow Pipelines.

The exporter should provide visibility into pipeline executions, scheduling, failures, resource usage, and operational health.

---

## Why

Current Kubernetes monitoring focuses on infrastructure.

There is very little observability around Kubeflow Pipelines themselves.

Platform engineers should be able to answer questions like:

- Which pipelines are currently running?
- Which pipelines fail most often?
- Average execution duration.
- Concurrent executions.
- Scheduled pipeline statistics.
- Namespace level activity.
- Historical trends.

---

## Long Term Goal

Become a useful open source companion project for Kubeflow.