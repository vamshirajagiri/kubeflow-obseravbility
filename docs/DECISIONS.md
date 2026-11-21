# Architectural Decision Records

Every major decision should be recorded here.

Use this format:

- Status
- Context
- Decision
- Why
- Tradeoffs
- Consequences

## ADR-001: Build An Exporter Instead Of Modifying Kubeflow

Status: Accepted

Context:

Kubeflow Pipelines does not currently expose all pipeline-level operational
signals this project wants in an easy Prometheus-ready form.

Decision:

Build a separate Prometheus exporter.

Why:

An exporter lets platform teams add observability without patching or rebuilding
Kubeflow itself.

Tradeoffs:

- good: easier to deploy independently
- good: can evolve faster than upstream Kubeflow
- bad: one more component to run
- bad: must track Kubeflow API compatibility

Consequences:

The project must be careful about API usage, auth, deployment, and metric
stability.

## ADR-002: Prefer Kubeflow Pipelines API Over Direct Database Access First

Status: Accepted

Context:

KFP run data exists in several places:

- public REST API
- backend database tables such as `run_details`
- Kubernetes and Argo workflow resources

Direct database access exposes rich fields, but it couples this exporter to
internal schema details and requires database credentials.

Decision:

Use the Kubeflow Pipelines REST API as the first source of truth for exporter
metrics.

Why:

The API is the public contract. It already includes namespace filtering,
pagination, auth behavior, run states, timestamps, and run details. A production
exporter should start from the least surprising integration point.

Real-world example:

If a platform team upgrades Kubeflow and the database schema changes, an
API-based exporter has a better chance of continuing to work. A direct database
exporter can break even if the UI and SDK still work.

Tradeoffs:

- good: simpler deployment
- good: respects KFP auth and namespace isolation
- good: less coupling to KFP internals
- bad: may expose less detail than the database
- bad: may need pagination and caching to avoid API pressure
- bad: some metrics may require extra API calls

Consequences:

The next design step should define the exact API calls, polling interval,
pagination strategy, and metric labels before implementation.

## ADR-003: Target Kubeflow Pipelines v2 First

Status: Accepted

Context:

Kubeflow Pipelines v1 and v2 differ in API shape, object model, and deployment
behavior. Supporting both immediately would force the project to design around
compatibility before it has a working core product.

Decision:

Target Kubeflow Pipelines v2 first.

Why:

The project should build one correct, understandable exporter before adding
compatibility layers. KFP v2 is the better first target because active users are
expected to move toward it, and the v2beta1 API exposes runs with fields such as
state, timestamps, pagination, and namespace filtering.

Real-world example:

If we support v1 and v2 from day one, every collector starts asking questions
like "which API shape am I reading?" and "does this state mean the same thing?"
That creates abstractions before the product has proven its basic behavior.

Tradeoffs:

- good: simpler first implementation
- good: cleaner docs and metric semantics
- good: faster path to a usable product
- bad: KFP v1 users are not supported at first
- bad: later v1 support may require compatibility work

Consequences:

Docs, metrics, and code should assume KFP v2 unless a section explicitly says
otherwise.

## ADR-004: Keep Default Metric Labels Low Cardinality

Status: Accepted

Context:

Prometheus stores one time series for every unique combination of metric name
and labels. Labels with values that grow forever create high cardinality.

Decision:

Default metrics must avoid unbounded labels such as run ID, run display name,
workflow name, pod name, user email, and arbitrary pipeline parameters.

Why:

The exporter should help operate Kubeflow, not create a new monitoring scaling
problem.

Real-world example:

`namespace` and `state` are safe labels because their value sets are small.
`run_id` is unsafe because every new pipeline run creates another label value.
On a busy platform, that can create thousands or millions of time series.

Tradeoffs:

- good: safer Prometheus resource usage
- good: easier dashboards and alerts
- bad: less per-run detail in default metrics
- bad: some debugging use cases may require logs or optional debug endpoints

Consequences:

The first metric contract prefers grouped operational metrics. Per-run identity
should not appear in default Prometheus labels.

## ADR-005: Do Not Expose Historical Run Counters In v0

Status: Accepted

Context:

Counters only go up during one process lifetime. Kubeflow stores historical
runs, and the exporter may read the same completed run more than once across
polls or restarts.

Decision:

Do not expose metrics such as `kubeflow_pipelines_runs_total` or
`kubeflow_pipelines_failed_runs_total` in v0.

Why:

Without durable state or event ingestion, the exporter cannot safely know
whether it has already counted a historical completed run.

Real-world example:

If the exporter restarts, lists the last 1,000 completed runs, and increments a
counter for each one again, Prometheus sees fake new failures. That creates bad
alerts and destroys trust in the exporter.

Tradeoffs:

- good: avoids misleading counters
- good: keeps first version honest
- bad: users do not get total historical run counters immediately

Consequences:

v0 should focus on current state, exporter health, API request behavior, and
data freshness. Historical counters can be revisited after we have a reliable
deduplication strategy.

## ADR-006: Default To All KFP-Visible Namespaces

Status: Accepted

Context:

The exporter talks to the Kubeflow Pipelines API, not directly to the Kubernetes
API. KFP already knows which pipeline runs, experiments, and recurring runs
belong to KFP namespaces.

Decision:

The default exporter mode should collect all KFP-visible data returned by the
KFP API for the exporter identity. Namespace filtering should be optional.

Why:

Platform engineers usually want one exporter that observes Kubeflow Pipelines
as a platform service. Requiring every namespace to be listed in configuration
creates operational work and makes new team onboarding noisy.

Real-world example:

If a platform has 40 ML namespaces, an allow-list-first exporter forces the
operator to update config every time a team is added. A KFP-visible-by-default
exporter starts observing the new KFP namespace when KFP exposes it to the
exporter identity.

Tradeoffs:

- good: simpler default deployment
- good: less repeated configuration
- good: one exporter can serve platform-wide monitoring
- bad: operators who need narrower visibility must configure namespace filters
- bad: we must clearly document that "all visible" means KFP-visible, not every
  Kubernetes namespace

Consequences:

The first implementation should support no namespace filter by default and an
optional configured namespace list for restricted collection.
