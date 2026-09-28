# Product Price Tracker: Project Plan

## Purpose and scope

Build a general-purpose Product Price Tracker that records retailer listing prices over time and helps users understand current prices, historical changes, promotions, and alert conditions. The tracker should make the evidence behind a price understandable, including where it came from, when it was observed, and which conditions apply.

A Product describes a retailer-independent item. A Listing describes a specific retailer product page. Early phases track explicitly selected listings; automatic cross-retailer product matching comes later. A Provider is an integration that collects data from a retailer or other permitted source.

These documents describe intended behavior, not existing functionality. Task 0 adds documentation only. The repository initially contains a single heading in `README.md` and no application infrastructure.

## Core goals

- Collect trustworthy, timestamped prices and availability from multiple providers over time.
- Preserve immutable observations so history remains auditable.
- Show current known prices with their freshness, source, currency, and stock state.
- Explain promotions and conditional effective prices without presenting potential savings as guaranteed.
- Notify users when an explicitly defined price condition is met.
- Add providers without coupling their failures or retailer-specific behavior to the core domain.
- Prefer a small, maintainable system with clear tests and operational visibility.

## Feature scope

### Prices and availability

Store retailer list price and sale price as separate observed fields where the source provides them. Store MSRP only when it is explicitly identified as manufacturer-suggested pricing, with provenance. Never infer MSRP from an unlabeled crossed-out price.

Use integer minor units and an explicit currency for every monetary amount. Support at least `in_stock`, `out_of_stock`, and `unknown` stock states. Show missing or stale information honestly; missing does not mean zero, and a collection failure does not mean out of stock.

### Price history

Keep immutable PriceObservations with listing, observation time, source, and price meaning. Offer bounded historical queries and a chart that labels the plotted price series and currency. Preserve gaps and stock changes rather than inventing continuous availability or observations.

### Promotions and effective price

Represent coupons, cashback, membership requirements, validity windows, and eligibility conditions explicitly. Keep those facts separate from an EffectivePrice calculation. Display the calculation assumptions and distinguish checkout reductions from later or uncertain cashback.

Begin with a limited, documented set of promotion rules. Unknown eligibility, stacking, or validity must not silently produce a guaranteed effective price. More complicated offers can remain descriptive until supported.

### Alerts

Introduce listing-level PriceAlerts with explicit thresholds, currencies, price bases, and stock/freshness rules. Evaluate against trustworthy observations and suppress duplicate notifications. Conditional EffectivePrice alerts require an explicit eligibility scenario; ordinary price alerts must not silently include coupons or cashback.

The delivery channel and user ownership model will be selected before alert delivery is implemented. Reliable delivery and visible failures matter more than supporting many channels.

## Product principles

1. **Correctness and data trust before feature count.** A qualified or unknown value is better than a misleading price.
2. **Evidence before inference.** Preserve observed facts and identify derived values and their inputs.
3. **Meaning before presentation.** MSRP, retailer list price, sale price, coupon, cashback, membership conditions, and effective price are different concepts.
4. **Freshness is visible.** The most recent known price is not necessarily a live price.
5. **Failures stay local.** A broken provider should not stop other providers or make historical data inaccessible.
6. **Incremental delivery.** One small task with acceptance criteria should normally produce one small, manually reviewed commit.
7. **Simple infrastructure first.** Add operational components only when evidence shows the existing design is insufficient.

## Initial technology stack

- **Frontend:** Next.js, with TypeScript, for the user interface.
- **Backend:** Go for domain logic, APIs, providers, and collection jobs.
- **Database:** PostgreSQL for persisted domain data, observations, and initial operational state.
- **Repository:** One repository and one Go codebase; the API and collector may have separate entry points/binaries.

Exact versions, routing libraries, database drivers, migration tooling, chart libraries, and deployment choices are deferred to the tasks that require them. A local PostgreSQL environment is planned after repository bootstrap; Task 0 creates no setup files, dependencies, or Docker files.

## Explicit early non-goals

- Automatic product discovery at web scale or automatic cross-retailer product matching.
- Universal support for every retailer, product variant, or promotion type.
- A universal delivered-cost calculation covering tax, shipping, geography, and all checkout rules. Any exclusions must be visible.
- Currency conversion or ranking prices across different currencies as though they were directly comparable.
- Purchase automation, checkout, payment processing, or inventory reservation.
- Bypassing authentication, anti-bot controls, or source access restrictions.
- Predictive pricing, recommendations, forecasting, or elaborate analytics.
- A full account, billing, multi-tenant, or mobile-app platform. Ownership and access controls must be addressed before a deployment needs them.
- Kafka, Redis, microservices, Kubernetes, Elasticsearch, or distributed orchestration without a measured need.

## Phased roadmap

1. **Documentation and repository bootstrap:** establish the planning baseline, minimal Go/Next.js structure, local PostgreSQL access, and repeatable migrations.
2. **Domain and persistence:** define core semantics, persist products/listings and immutable observations, and expose narrow catalog APIs.
3. **Providers and ingestion:** introduce a provider contract, deterministic fake provider, validated ingestion, failure isolation, and a first real provider.
4. **Collection and current-price MVP:** schedule bounded collection, query current known prices, and display listings, freshness, and stock in a minimal frontend.
5. **History:** expose bounded price history and a labeled chart without hiding missing data.
6. **Promotions and price meaning:** preserve promotion facts, calculate limited scenario-based effective prices, and validate MSRP-specific display and comparisons. MSRP semantics apply from the first domain implementation.
7. **Alerts:** add alert configuration, evaluation, and a reliable initial notification channel.
8. **Additional providers and reliability:** prove extensibility, strengthen retry behavior, visibility, backup/recovery, and operational checks. Basic isolation and correctness are required earlier, not postponed to this phase.
9. **Later product matching:** propose cross-retailer matches with variant evidence and review before expanding automation.

See [TASKS.md](TASKS.md) for the ordered, commit-sized work queue. See [DOMAIN_MODEL.md](DOMAIN_MODEL.md) for terminology, [ARCHITECTURE.md](ARCHITECTURE.md) for boundaries, and [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md) for the mandatory review process. Update the relevant documents when reviewed decisions change these plans.
