# Architecture

## Initial system

Use a Next.js frontend, a Go backend, and PostgreSQL in one repository. This is a planned architecture; no application setup is part of Task 0.

```text
Browser -> Next.js frontend -> Go API -> PostgreSQL
                                          ^
                                          |
                              Go collector / ingestion
                                          |
                                   Provider adapters
                                          |
                               Retailers / source APIs
```

The frontend renders catalog, current prices, history, promotion explanations, alert controls, and triggered events. The Go backend owns validation, domain rules, persistence access, and derived-price calculations. PostgreSQL is the initial durable store. The frontend does not query the database directly or implement a second copy of price rules.

## Codebase and process boundaries

The API and collector may run as separate Go binaries while sharing one codebase and domain packages. This allows collection work to have a separate lifecycle and resource budget from user-facing requests without introducing independent services, network protocols, or repositories.

Keep logical boundaries for domain rules, application operations, persistence, HTTP handling, and providers. Package structure should follow actual needs rather than a large framework prepared in advance. Do not implement every future boundary during bootstrap.

User-facing reads use persisted data; they should not need a live retailer request. Start with a single collector/scheduler instance. Introduce coordination only if overlapping work or multiple instances require it.

## Provider abstraction

A Provider adapts one source to the collection contract. It may use an official API or another permitted collection method. A Retailer is a business/domain identity; it is not the same thing as a Provider implementation, and their relationship need not always be one-to-one.

The contract should accept a listing/source reference and cancellation/deadline context, then return normalized observed facts with provenance or an explicit collection error. Exact Go interfaces will be defined in the provider task.

Provider responsibilities:

- Fetch source data within configured access and rate limits.
- Parse prices, currency, stock state, and supported promotion facts without inventing missing values.
- Identify source references, observation time, and adapter identity/version where needed for diagnosis.
- Distinguish unavailable fields from collection errors and expose meaningful error categories.

Shared ingestion responsibilities:

- Validate results against domain invariants and listing identity.
- Persist complete, accepted observations atomically.
- Handle delivery retries without duplicating the same collection result.
- Record collection outcomes separately from pricing facts.

Providers do not write directly to the database or define shared EffectivePrice rules. A deterministic fake provider proves the contract before a real integration is introduced. Fixture-based tests keep routine verification independent of live retailer availability.

## Failure isolation

Bound provider calls with timeouts, cancellation, limited concurrency, and source-specific rate limits. Contain failures per collection attempt/listing and continue unrelated work; apply provider-level backoff when failures affect a whole source. Keep API reads independent of provider health.

A failed request, parser error, or blocked source produces an operational failure record, not a fabricated zero price or out-of-stock PriceObservation. Keep the last accepted observation available with its original timestamp and visible freshness. Do not let an old price appear current merely because a retry occurred.

Use bounded retries for transient failures and avoid immediate retry loops for persistent parsing or access failures. More advanced isolation, such as a circuit breaker or separate provider workers, is justified only by recurring failure patterns. Establish basic isolation before the first real provider; expand diagnostics and recovery in reliability tasks.

## Observed facts and derived data

PriceObservations preserve what a source reported at a particular time. Promotion terms and explicit MSRP claims also require provenance. Collection success/failure metadata is operational data and should remain distinct from market observations.

EffectivePrice, current-price selection, historical aggregates, discount percentages, and alert decisions are derived results. Each needs defined inputs and semantics. Calculations must not overwrite their source observations. Persist derived results only when useful, along with enough input/rule identity to explain or reproduce them.

Alert evaluation runs after ingestion commits an observation, using a separate transaction. Evaluation failure is logged without changing ingestion success or observation facts. PostgreSQL stores alert configuration and immutable triggered events, unique per alert/observation pair. Frontend controls and event reads are bounded. This MVP has no evaluation retry worker or historical backfill. External notifications, delivery attempts/retries, queues, and notification infrastructure are outside project scope.

Current-price queries select a coherent accepted observation by documented observation-time and tie-breaking rules. Do not combine a fresh sale price with an old list price or promotion without explicit provenance. An observation with a missing price must not silently inherit an older price as if newly observed. If a last-known price is shown, retain its separate timestamp and label.

Use integer minor units and explicit currency throughout database, Go, and API representations. See [DOMAIN_MODEL.md](DOMAIN_MODEL.md) for the authoritative definitions, rounding expectations, and examples.

## Persistence and API direction

Use relational constraints and migrations for stable identities and relationships. Observations are append-only in normal application behavior. Use transactions where partial writes would create misleading data. Add indexes based on the initial query shapes, especially listing history and current-price retrieval.

Observation invalidation is a separate append-only record with reason/time; it never changes original facts. Current prices, public history, and subsequent alert evaluation exclude invalidated observations. A bounded audit API retains visibility of all original facts and invalidation metadata. Invalidation and alert evaluation serialize on the Listing row so evaluation cannot consume an invalidation already committed for that Listing. Existing alert events and promotion evidence are never rewritten; historical decisions may reference subsequently invalidated facts. See [RELIABILITY.md](RELIABILITY.md).

Collection attempt/success/error status and consecutive failure counts are process-local diagnostics and reset on restart. They are not persisted or restorable from a database dump. PostgreSQL backups preserve all durable catalog, observation, promotion, alert/event, invalidation, and Goose state using standard pg_dump/pg_restore; recovery is verified in separate disposable databases.

Keep APIs narrow and bounded: product/listing operations, current known prices, then historical ranges and alert operations. Define pagination, validation, error responses, and freshness semantics as each API is built. Do not choose a broad API framework or protocol solely for hypothetical future clients.

Store only source evidence needed for traceability; retaining every raw page is not an initial requirement. Avoid secrets and personal information in logs or evidence. Configure credentials outside tracked files. Access controls and public deployment requirements must be settled before exposing writable or user-specific APIs publicly.

## Infrastructure intentionally excluded initially

### Kafka and other message brokers

A small collector can fetch and persist directly, with durable collection state in PostgreSQL as needed. A broker would add delivery semantics, operational overhead, and more failure modes before there are independent consumers.

Reconsider when measured ingestion volume, buffering during prolonged downstream outages, replay requirements, or multiple independent consumers exceed the simple design. A broker would not remove the need for idempotency and transaction boundaries.

### Redis

Start with PostgreSQL queries, suitable indexes, and bounded workers. A separate cache or coordination store would add invalidation and consistency concerns.

Reconsider if measured latency/traffic demonstrates a cache need that query improvements cannot address, or multiple instances need shared rate limiting or short-lived coordination. PostgreSQL remains the source of truth, and cache freshness must be explicit.

### Microservices

One Go codebase with optional API/collector binaries provides useful separation without distributed transactions, duplicated deployment machinery, or service-to-service contracts.

Reconsider independent services when clear ownership boundaries, sustained scaling differences, or a demonstrated need for stronger fault isolation justify the operational cost. A new provider alone is not sufficient reason.

### Kubernetes

Early development and deployment do not require a cluster scheduler. Select a simple hosting approach once deployment requirements are known.

Reconsider if the system has enough independently operated workloads, availability requirements, and operational support to benefit from cluster orchestration, or an existing supported platform makes it simpler.

### Elasticsearch and dedicated search infrastructure

Initial product/listing retrieval and limited search should fit relational queries. A search cluster would require another index, synchronization, and consistency policy.

Reconsider when measured catalog size, relevance requirements, typo tolerance, or complex search workloads exceed PostgreSQL's practical capabilities. Product matching still requires domain evidence; search infrastructure does not establish product identity.

### Other distributed infrastructure

Do not add a workflow engine, data warehouse, sharding, or specialized time-series database preemptively. Reconsider individual components when observed data volume, retention, recovery, or analytics workloads establish a concrete requirement. Document the limitation, simpler alternatives, and migration cost before adoption.
