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

Acceptance: compare a manageable candidate set using official access, price/identity/currency capability, historical-retention rights, rate limits, structured data stability, and product/listing fit. Record PROMISING, BLOCKED, or NEEDS CLARIFICATION with evidence; rank only PROMISING candidates. Complete a focused access/retention/model-fit gate before selecting an implementation target. Produce documentation only; no provider implementation. See the [candidate sweep and feasibility decisions](PROVIDER_RESEARCH.md#provider-candidate-sweep--task-17).

### Task 14: Best Buy access feasibility

Acceptance: document official access requirements and historical-retention constraints; explicitly approve, block, or require external clarification. Documentation only.

### Task 15: Dell access feasibility

Acceptance: assess official access, pricing data, and long-term historical storage; record the evidence and feasibility outcome. Documentation only.

### Task 16: eBay access feasibility

Acceptance: assess production access, retention rights, and narrow marketplace model fit; distinguish facts from unresolved questions. Documentation only.

### Task 17: Provider candidate sweep

Acceptance: compare a manageable set of official sources, classify retention/access feasibility, and rank only promising candidates. Documentation only.

### Task 18: France fuel feed domain-fit feasibility

Acceptance: evaluate catalog mapping, exact money precision, timestamps, and reuse rights against the existing domain; record the decision without changing code.

These user-directed research tasks supersede the original task numbering. The first
real adapter and its ingestion validation remain deferred until a source passes
access, retention, and domain-fit gates. Then schedule two small tasks: fixture-based
adapter implementation, followed by permitted ingestion validation. No research
outcome implicitly authorizes implementation; see [PROVIDER_RESEARCH.md](PROVIDER_RESEARCH.md).

## Current prices and frontend MVP

### Task 19: Collection runtime and current-price backend MVP

Acceptance: run a bounded explicit set of active Listings on a configurable, non-overlapping schedule through existing batch ingestion; isolate failures and stop cleanly. Expose separate collection status and coherent latest observations through Listing/Product price endpoints, with deterministic ties, freshness, missing-price handling, and conservative same-currency comparisons. Fake is sufficient; no real provider, frontend, or promotions. Verify pure unit tests and disposable-database integration tests.

### Task 20: Frontend MVP and price history

Acceptance: display API-backed products, current Listing prices, stock, freshness, and source links with loading, empty, and error states. Add bounded historical reads for 1D/1W/1M/3M/1Y/ALL, deterministic ordering, conservative historical-low/change metadata, and a selectable history chart that preserves missing points and currency distinctions. Verify backend unit/integration tests and frontend parsing, formatting, lint, and build. No real provider or create/edit forms.

### Task 23: Catalog Management and Collection Controls

Acceptance: reuse catalog APIs for Product/Retailer/Listing creation and bounded browsing with clear relationships, success/error states, and refreshed frontend data. Display process-local collection status separately from observation freshness. Add a synchronous single-Listing manual trigger through the configured runtime/ingestion path, prevent overlapping attempts for that Listing, and report unavailable providers honestly. Verify forms, conflicts, status semantics, collection success/failure, and existing functionality. No edit/delete, new scheduler, provider implementation, or schema changes.

## Promotions, effective price, and MSRP

### Task 21: Promotions and EffectivePrice MVP

Acceptance: append narrow fixed/percentage/cashback/membership promotion evidence with provenance, validity, and explicit unknown conditions; calculate explainable selected scenarios using exact arithmetic while separating payable and potential cashback/net. Add minimal evidence/scenario APIs and frontend display, with conditional/unavailable states, unit/integration tests, migration rollback/reapply, and frontend checks. No provider parsing, discovery, alerts, or generalized rules engine.

### Task 24: Reliability, data quality, and backup/restore

Acceptance: append reason/time invalidation records while retaining immutable observation facts; exclude invalid observations from current/history/alert evaluation and expose bounded audit/invalidation controls. Preserve historical promotion/event evidence, add safe diagnostic logs and process-local failure counts, and verify Docker pg_dump/pg_restore against disposable seeded databases including Goose metadata. Review existing query/index shapes without speculative infrastructure. The former promotion-persistence task was covered by Task 21.

### Task 25: Promotion collection for one provider

Acceptance: normalize a narrow promotion type from fixtures into preserved evidence through ingestion. Unsupported or ambiguous terms do not become invented executable discounts.

### Task 26: Covered by Task 21

The narrow EffectivePrice calculation is included in Task 21; broader combinations require a separate reviewed task.

### Task 27: Covered by Task 21

The selected-scenario API and explanation are included in Task 21.

### Task 28: Covered by Task 21

The separate promotion and derived-result display is included in Task 21. Task 25 provider collection remains deferred; subsequent scope is unchanged.

### Task 29: MSRP provenance and comparisons

Acceptance: validate explicit MSRP evidence end to end and label any MSRP-based comparison separately from retailer-list or historical comparisons. Never infer MSRP from an unlabeled crossed-out price. Reinforce semantics already required since Task 4.

## Alerts

### Task 22: Price Alerts MVP

Acceptance: support target-price, percentage-drop, and historical-low alerts on observed offer/sale prices; persist configuration and durable events. Evaluate after successful ingestion without invalidating observations on evaluation failure. Enforce currency, missing-price, stock, and first-observation rules; suppress duplicate alert/observation events while allowing later qualifying events. Add bounded APIs and frontend create/list/enable/disable/event views. Verify domain, ingestion, persistence, API, frontend, and migration behavior. Promotions/EffectivePrice, external delivery, auth, and generic rule engines are excluded.

### Tasks 30–32 and 34: Covered by Task 22

The combined MVP includes alert model/persistence, configuration API, evaluation, and frontend controls.

### Task 33: Removed from project scope

There is no external notification delivery: no email, SMS, push, webhooks, delivery attempts, or delivery retries. Triggered events are stored and viewed in the application only.

## Additional providers and reliability

### Task 35: Second provider

Acceptance: add one independently tested provider through the existing contract, with documented supported contexts and limitations. Verify failures remain isolated; revise the abstraction only for demonstrated differences.

### Task 36: Collection retry and rate-limit hardening

Acceptance: refine backoff and source-specific rate behavior using actual provider failure categories; verify restart and repeated-failure behavior without unbounded retries.

### Task 37: Operational visibility

Initial scope covered by Task 24: collection/alert-evaluation outcomes, timing and failure diagnostics, a health-check procedure, and safe logs. Broader monitoring requires a separate task.

### Task 38: Data-quality handling

Covered by Task 24: invalidation without modifying original facts, explicit query/alert treatment, and retained historical evidence.

### Task 39: Backup and restore verification

Covered by Task 24: verify durable application and Goose state using disposable backup/restore. Process-local collection diagnostics intentionally reset on restart.

### Task 40: Query and retention review

Task 24 reviews representative query/index shapes; larger-data measurement remains deferred until a demonstrated need. Do not delete historical data or add new infrastructure without an explicit reviewed decision.

## Later product matching

### Task 41: Matching requirements and evidence

Acceptance: document supported identity/variant criteria and representative positive/negative examples for cross-retailer matching. Define review and correction needs before implementing automatic associations.

### Task 42: Match suggestions

Acceptance: propose candidate matches with explainable evidence against a small labeled sample; leave Product/Listing associations unchanged until reviewed.

### Task 43: Reviewed match application

Acceptance: apply approved associations with an audit trail and a correction path; preserve listing identities and immutable observation history. Verify materially different variants are not silently merged.

## Deferred expansion

Add further provider, deployment, access-control, matching, or scale tasks only when concrete requirements exist. Repeat small provider tasks individually rather than combining many integrations into one commit. Public deployment requires its own reviewed access and operational readiness work; this roadmap does not imply the MVP is ready for unrestricted exposure.
