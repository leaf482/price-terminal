# Ordered Task Roadmap

## How to use this roadmap

Execute tasks in order unless review explicitly revises the sequence. Each task should normally fit one small commit and must receive a concise implementation prompt with acceptance criteria. Codex does not create commits; the user commits manually only after ChatGPT review passes. See [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md).

The criteria below are intentionally short. Refine each against the current repository before starting, and split it further if needed. They do not authorize future work or select unnecessary implementation details. All price-related work follows [DOMAIN_MODEL.md](DOMAIN_MODEL.md).

## Documentation and foundation

### Task 0: Documentation bootstrap

Acceptance: inspect the repository; create the five planning documents covering purpose, architecture, domain, workflow, and roadmap; verify terminology and scope. No application code, setup, Docker files, dependencies, or commit.

### Task 1: Repository bootstrap

Acceptance: establish a minimal Next.js frontend and Go backend layout in one repository; document their local commands; run the relevant build/check commands. No provider, database, or business feature implementation.

### Task 2: Local PostgreSQL environment

Acceptance: document and verify a reproducible local PostgreSQL connection with configuration examples that contain no secrets. Choose the local setup method in this task; no application schema yet.

### Task 3: Migration infrastructure

Acceptance: select a minimal migration approach, document commands, and verify migration execution/state tracking on a disposable database. Domain tables remain for later tasks.

## Core domain and persistence

### Task 4: Core domain types and invariants

Acceptance: define the initial Product, Retailer, Listing, PriceObservation, money/currency, and stock concepts; test invalid amounts/relationships and price distinctions. Preserve MSRP semantics now; defer promotion and alert behavior.

### Task 5: Catalog persistence

Acceptance: migrate and persist Product, Retailer, and Listing records with relationship and duplicate-identity safeguards; verify round trips and constraint failures against PostgreSQL.

### Task 6: Observation persistence

Acceptance: migrate and append PriceObservations with provenance and distinct price fields; verify historical preservation, currency/stock handling, and retry identity without overwriting observations.

### Task 7: Product APIs

Acceptance: add minimal product create/read/list operations with validation, bounded listing, and consistent error responses; verify valid and invalid requests.

### Task 8: Retailer and listing APIs

Acceptance: support the minimum retailer registration and listing create/read/list flow; validate product/retailer references and duplicate source context. Verify invalid references and bounded reads.

## Providers and collection

### Task 9: Provider abstraction

Acceptance: define a small collection contract with source reference, cancellation, observed results, and errors; keep database access and effective-price rules outside providers. Verify contract validation.

### Task 10: Fake provider

Acceptance: implement deterministic fixtures for normal pricing, unavailable fields, stock states, and failures. Tests run without external retailer access.

### Task 11: Single-listing ingestion

Acceptance: collect one listing through the provider contract, validate the result, and persist it atomically. Verify retried delivery does not duplicate an observation and later unchanged observations remain distinct.

### Task 12: Provider failure isolation

Acceptance: bound collection calls and isolate failures across listings/providers; record failures separately from observations. Verify a failing source does not block a healthy source or erase its last accepted data.

### Task 13: First real provider research/selection

Acceptance: evaluate an official API or permitted collection method, source policy/access constraints, rate limits, structured data stability, supported product/listing context, and known limitations. Produce a documented provider decision only; no provider implementation.

### Task 14: First real provider adapter

Acceptance: implement the narrow selected provider with fixture-based parsing and tests for supported price and stock facts. Expose limitations and fail clearly on malformed data.

### Task 15: First real provider ingestion validation

Acceptance: connect the adapter to the existing single-listing ingestion path; validate a representative permitted collection and persisted semantics, including provenance and unavailable fields. Keep routine tests fixture-based.

### Task 16: Collection scheduling

Acceptance: schedule tracked listings at a configurable interval with bounded work and defined overlap/restart behavior for one collector instance. Verify no unintended concurrent duplicate collection and clean shutdown.

## Current prices and frontend MVP

### Task 17: Current-price queries

Acceptance: return a coherent latest accepted observation with currency, stock, timestamp, and freshness; define out-of-order/tie behavior. Verify missing prices, no history, and stale observations without silently merging old fields.

### Task 18: Frontend catalog view

Acceptance: display API-backed products and their listings with loading, empty, and error states. Keep layout minimal and verify navigation against representative data.

### Task 19: Frontend current-price view

Acceptance: show distinct observed prices, currency, source link, stock, and freshness on a listing view; avoid misleading zero/default prices. Verify unknown and stale states.

## Historical prices

### Task 20: Historical price API

Acceptance: expose bounded listing observations with stable ordering and time-range handling; preserve semantic price fields, currency, and stock. Verify range boundaries and empty results.

### Task 21: Historical price chart

Acceptance: chart a clearly labeled price series using the history API; expose timestamps and currency, and show gaps/unknown states honestly. Verify one-point, empty, and interrupted histories.

## Promotions, effective price, and MSRP

### Task 22: Promotion evidence model and persistence

Acceptance: store listing applicability, provenance, validity, and supported conditions while preserving historical evidence. Keep coupon, cashback, and membership concepts distinct; test unknown conditions.

### Task 23: Promotion collection for one provider

Acceptance: normalize a narrow promotion type from fixtures into preserved evidence through ingestion. Unsupported or ambiguous terms do not become invented executable discounts.

### Task 24: Effective-price calculation

Acceptance: calculate a limited documented scenario from explicit observation/promotion inputs using exact arithmetic and tested rounding; separate checkout reductions from cashback and reject unsupported combinations.

### Task 25: Effective-price API and explanation

Acceptance: expose derived results with input references, assumptions, exclusions, and conditional/unavailable states. Verify results remain distinct from observed prices.

### Task 26: Promotion and effective-price display

Acceptance: show applicable terms and calculation explanations without advertising conditional savings as universal. Verify unknown eligibility and potential cashback displays.

### Task 27: MSRP provenance and comparisons

Acceptance: validate explicit MSRP evidence end to end and label any MSRP-based comparison separately from retailer-list or historical comparisons. Never infer MSRP from an unlabeled crossed-out price. Reinforce semantics already required since Task 4.

## Alerts

### Task 28: PriceAlert model and persistence

Acceptance: persist listing target, threshold/currency, price basis, stock/freshness rules, and enabled state; define inclusive threshold and re-arm behavior. Test invalid and ambiguous configurations.

### Task 29: Alert configuration API

Acceptance: expose minimal alert configuration operations with validation and explicit ownership/access assumptions for the intended deployment. Choose the initial delivery destination model before delivery work.

### Task 30: Alert evaluation

Acceptance: evaluate alerts against eligible observations with traceable decisions and duplicate suppression. Verify stale/missing data, stock rules, repeated collections, and conditional price scenarios.

### Task 31: First notification delivery channel

Acceptance: deliver evaluated alert events through one selected channel with durable attempt state, bounded retries, and documented duplicate-delivery handling. Test failures without sending real notifications.

### Task 32: Alert frontend controls

Acceptance: create, view, and disable alerts through the API; display threshold basis, conditions, and useful delivery status. Verify validation and error states.

## Additional providers and reliability

### Task 33: Second provider

Acceptance: add one independently tested provider through the existing contract, with documented supported contexts and limitations. Verify failures remain isolated; revise the abstraction only for demonstrated differences.

### Task 34: Collection retry and rate-limit hardening

Acceptance: refine backoff and source-specific rate behavior using actual provider failure categories; verify restart and repeated-failure behavior without unbounded retries.

### Task 35: Operational visibility

Acceptance: expose enough collection/delivery outcomes and timing information to diagnose failed or stale listings; document a small health-check procedure and verify logs omit secrets.

### Task 36: Data-quality handling

Acceptance: add a minimal way to identify invalid accepted observations without modifying original facts, and define query/alert treatment. Verify traceability and historical promotion/calculation references remain intact.

### Task 37: Backup and restore verification

Acceptance: document and exercise database backup/restore on disposable data; verify products, listings, observations, and operational state needed for safe restart survive.

### Task 38: Query and retention review

Acceptance: measure representative current/history queries, address demonstrated bottlenecks with small changes, and document retention assumptions. Do not delete historical data or add new infrastructure without an explicit reviewed decision.

## Later product matching

### Task 39: Matching requirements and evidence

Acceptance: document supported identity/variant criteria and representative positive/negative examples for cross-retailer matching. Define review and correction needs before implementing automatic associations.

### Task 40: Match suggestions

Acceptance: propose candidate matches with explainable evidence against a small labeled sample; leave Product/Listing associations unchanged until reviewed.

### Task 41: Reviewed match application

Acceptance: apply approved associations with an audit trail and a correction path; preserve listing identities and immutable observation history. Verify materially different variants are not silently merged.

## Deferred expansion

Add further provider, deployment, access-control, matching, or scale tasks only when concrete requirements exist. Repeat small provider tasks individually rather than combining many integrations into one commit. Public deployment requires its own reviewed access and operational readiness work; this roadmap does not imply the MVP is ready for unrestricted exposure.
