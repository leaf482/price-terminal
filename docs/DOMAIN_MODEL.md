# Domain Model

## Shared terminology and rules

This document defines concepts and invariants, not a final database schema or exhaustive field list. Missing values are unknown or unavailable, never implicitly zero. Observed facts describe a source claim; they are not a guarantee of what a particular buyer will pay.

### Money and currency

- Represent money as an integer number of minor units plus an explicit currency code; never use binary floating-point for monetary values or calculations.
- Interpret minor units using the currency's defined exponent, not a universal assumption of two decimal places. For example, USD 19.99 is `1999` minor units.
- Parse source decimal strings exactly. Validate currency agreement, valid ranges, and arithmetic overflow.
- Use exact decimal/rational arithmetic or scaled integers for percentage rules; define and test rounding when producing minor-unit amounts.
- Never compare or combine amounts of different currencies without a separately defined conversion policy. Currency conversion is an early non-goal.
- Preserve missing vs explicitly observed zero. A zero price requires valid source evidence, not a parsing default.

### Price meanings

- **MSRP:** a manufacturer-suggested retail price, supported by an explicitly identified claim and source. It is not automatically the retailer's previous price or an item-wide timeless value.
- **Retailer list price:** the reference/list price presented by a retailer. It may be a comparison price and does not prove the item previously sold for that amount.
- **Sale price:** a current sale/offer price explicitly identified by the source. Do not label every current price as a sale when the source does not support that meaning.
- **Coupon:** a conditional discount, such as a code or clipped offer. It is a Promotion, not a replacement for the observed sale price.
- **Cashback:** a potential later rebate with its own conditions. It does not reduce the immediate checkout price by default.
- **Membership conditions:** eligibility constraints that may apply to a price or promotion. They are not monetary savings by themselves.
- **Effective price:** a derived amount for a named scenario, using explicit inputs, eligibility, and calculation rules. It is not an observed retailer price.

If a source only provides a current offer amount without identifying list or sale meaning, preserve it as an observed offer price with that meaning rather than inventing either classification. Field names and storage details are deferred; distinct meanings must survive ingestion, storage, APIs, and display.

## Relationships

```text
Product 1 ---- many Listings many ---- 1 Retailer
Listing 1 ---- many PriceObservations
Listing 1 ---- many applicable Promotions (initial scope)
PriceObservation + Promotion evidence + scenario ----> EffectivePrice
Listing 1 ---- many PriceAlerts (initial scope)
```

Every Listing refers to one Product and one Retailer initially. Multiple listings can belong to the same retailer/product. Promotion applicability may later span multiple listings; the first implementation need only support explicit listing applicability. EffectivePrice is a derived concept and does not require its own database table initially.

## Product

**Responsibility:** represent a retailer-independent, specifically identifiable item or variant, with stable identity and descriptive attributes such as name, brand, model, and available identifiers.

A Product must not own a retailer URL, retailer-specific current price, or stock state. Material differences such as capacity, size, pack count, or condition must not be silently collapsed into one identity. Exact category-specific attributes can evolve when actual products require them.

Initial product/listing associations are explicit and reviewed. Matching by similar titles alone is insufficient. Later matching may propose associations with evidence and confidence; it must preserve listing and observation traceability.

## Retailer

**Responsibility:** identify the seller/store context in which listings exist, including the source identity and any supported market context needed to interpret prices.

A Retailer owns no universal Product identity. Market, currency, seller, or regional differences must not be silently combined. The first provider should declare its supported context; a full marketplace/seller model is deferred until required.

A Provider is collection infrastructure associated with sources; it is not a substitute for the Retailer domain entity.

## Listing

**Responsibility:** represent a retailer-specific product page for a defined item/variant and collection context.

A Listing links Product and Retailer and carries a canonical source URL and, where available, a retailer identifier. Avoid duplicate tracking of the same source/context. URL alone may not distinguish selected variants, so retain the source references needed to collect the intended item.

A Listing is the initial unit of collection, price history, and alerting. A displayed current price is selected from its observations; updating listing metadata must not rewrite historical observations. If a page changes to a different item, do not silently blend the two histories.

## PriceObservation

**Responsibility:** capture immutable historical facts accepted from a collection attempt for a Listing.

Preserve listing identity, source/provenance, observation time, ingestion time where needed, currency, observed price meanings, stock state, and relevant conditions/evidence. Represent timestamps as unambiguous instants, with UTC for storage/exchange. Distinguish when data was observed from when it was persisted; retries must not refresh the original observation time.

Required stock states are:

- `in_stock`: the source supports availability in the observation's context.
- `out_of_stock`: the source explicitly reports unavailability.
- `unknown`: availability was not determined from an otherwise accepted result.

A source can validly expose stock without a price or a price without known stock. A failed collection is separate operational data, not an observation that prices disappeared. Reject malformed results rather than silently converting parsing failures into absent fields.

Observations are append-only: no in-place correction of their historical facts. If an accepted observation is later found invalid, preserve it and record an explicit quality/invalidation annotation or superseding evidence; define whether queries exclude it. Normal application code must not erase history to fix a parser bug.

Retrying persistence of the same collection result must not append duplicates. A later, independent observation of an unchanged price remains a valid new observation because its time and freshness differ.

## Promotion

**Responsibility:** preserve a source-supported offer and its applicability and conditions separately from base price facts.

Promotion evidence can include type, source/reference, observed time, validity window, amount or rate, coupon code, minimum spend, caps, membership requirements, eligibility, and stacking terms where available. Unknown conditions remain unknown. Separate source-stated validity from when the tracker observed the offer.

Support only a small set of executable rules initially; unsupported offers may be recorded descriptively. Do not assume two promotions stack, that a buyer qualifies, or that cashback will be paid.

Keep historical promotion evidence stable through snapshots or versions so subsequent changes do not alter the meaning of old calculations. The storage mechanism is an implementation decision; reproducibility is the invariant.

## EffectivePrice

**Responsibility:** explain a derived price for a specific Listing observation and buyer/eligibility scenario.

Identify the base observation and chosen price basis, applicable promotion evidence, calculation time/rule identity, currency, assumptions, and exclusions. A useful result distinguishes immediate payable merchandise amount from potential net cost after delayed cashback. Tax and shipping are excluded unless explicitly included by a supported rule and supported data.

Unknown required inputs should produce an unavailable or clearly conditional result, not a confident numeric claim. Recalculation must not mutate PriceObservation or Promotion evidence. Persisted derived values, if introduced, must retain enough input references and rule information for explanation.

### Example

A listing reports MSRP USD 120.00, retailer list price USD 110.00, and sale price USD 90.00. It also advertises a USD 10.00 member coupon and USD 5.00 cashback. Stored amounts are respectively `12000`, `11000`, `9000`, `1000`, and `500`, each with currency USD and the correct semantic role.

For an eligible member, if evidence confirms these offers can be combined, the derived immediate merchandise amount is USD 80.00 and potential net cost after cashback is USD 75.00. Both exclude tax and shipping in this example. The observed sale price remains USD 90.00. Unknown eligibility or stacking prevents presenting USD 75.00 as an unconditional price.

## PriceAlert

**Responsibility:** describe an explicit condition to evaluate against a Listing's pricing data and track notification state separately from observed facts.

An initial alert includes a listing target, monetary threshold/currency, selected price basis, enabled state, stock and freshness policy, and an eventual owner/delivery destination. Specify whether the threshold is inclusive. Define repeat/re-arm behavior so repeated collections do not generate duplicate notifications for the same satisfied condition.

Default price basis and stock policy must be documented before implementation. For example, an alert can request an observed offer at or below USD 80.00 only while in stock and fresh. That example is not a silent global default. EffectivePrice-based alerts additionally require explicit scenario conditions.

Alert evaluation is derived behavior. Record which observation/calculation caused a decision and separate evaluation from delivery attempts so delivery can be retried safely. Missing prices, failed collection, expired promotions, or stale data must not trigger alerts by being interpreted as a bargain.
