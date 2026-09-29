# Provider access feasibility

Latest review: [eBay, Task 16](#ebay-access-feasibility--task-16).
The earlier Best Buy and Dell decisions are retained below for traceability.

## Best Buy access feasibility — Task 14

Checked: 2026-09-28. Outcome: **BLOCKED under the published API terms**.

This Task 14 feasibility review supersedes the conditional implementation handoff
in [the Task 13 decision](FIRST_PROVIDER_DECISION.md). The requested feasibility
gate takes precedence over the adapter work listed in [TASKS.md](TASKS.md).
No adapter is approved by this review; roadmap numbering is unchanged.

### Access: documented route versus verified issuance

The [developer portal](https://developer.bestbuy.com/) still advertises Get API
Key. Following its [login route](https://developer.bestbuy.com/login) in a browser
reached a Developer Portal-branded Best Buy sign-in/create-account page. The
create-account form displayed first name, last name, email, password and
confirmation, and mobile phone number, plus agreement notices for the site's
terms, privacy policy, and My Best Buy terms. This confirms a public onboarding
entry point, not successful developer enrollment or key issuance.

The API terms require a completed developer registration and accurate account
information. General Products access is not stated to require affiliate status;
Games/CD/DVD/Blu-ray content has separate affiliate/agreement restrictions.
[Developer Accounts and Keys; License Grant](https://developer.bestbuy.com/legal).

The [API guide](https://bestbuyapis.github.io/api-documentation/#getting-started)
describes email signup followed by emailed key-activation instructions.
Authentication uses the `apiKey` query parameter in HTTPS GET requests.

**Can a new credential actually be obtained today? Unverified.** No account was
created, registration submitted, activation email received, or authenticated
request made. Post-login developer form fields, approval criteria, and actual
issuance remain unknown. The current account UI is more involved than the guide's
email-only summary; neither proves issuance is operational. Do not report that
registration is closed, or that access has been granted.

### Data and exact endpoint

For an already identified US SKU, the documented Products endpoint supplies
lookup, price, and online availability together; no separate price/stock endpoint
is needed for this proposed scope:

```text
GET https://api.bestbuy.com/v1/products/{sku}.json?apiKey={key}&show=sku,url,regularPrice,salePrice,onSale,priceRestriction,onlineAvailability,orderable,specialOrder
```

`{sku}` and `{key}` are placeholders; this request was not executed.

- `salePrice`: current selling price; `regularPrice`: regular selling price.
- `onSale`: whether current price is below regular price.
- `onlineAvailability`: whether online purchase is possible; not inventory quantity.
- `orderable`/`specialOrder`: additional ordering context.
- `priceRestriction`: MAP/checkout restrictions; restricted pricing stays unsupported.
- No explicit MSRP field was established in the reviewed pricing dictionary.

These are documented capabilities, not verified live responses or permission for
historical use. See [response formats and selection](https://bestbuyapis.github.io/api-documentation/#response-format),
[pricing](https://bestbuyapis.github.io/api-documentation/#pricing), and
[availability](https://bestbuyapis.github.io/api-documentation/#availability).

Published Products quotas are **5 calls/second and 50,000/day per key**; quota
excess returns 403. [Operational Policy: Rate Limit](https://developer.bestbuy.com/legal).

### Historical retention: the blocking condition

The currently served terms are dated February 23, 2021. Their opening definition
includes all API-transmitted data in Content. Under Prohibited Uses, storage or
caching is allowed only temporarily, at most 72 hours, solely to improve display
response times. This is not limited to images or complete response documents.
No exception for normalized price/stock history is stated. The later retention
heading addresses backups, not an additional storage license. General Terms
requires modifications to be written and signed by an authorized representative.
[API terms](https://developer.bestbuy.com/legal).

The documentation's seven-day response-link expiration is not permission to
retain data seven days. [Response format](https://bestbuyapis.github.io/api-documentation/#response-format).

**Project determination:** permanent PriceObservations preserve source price and
stock facts, even after conversion to minor units and addition of our timestamp.
We cannot treat this normalization as an exemption. Long-term observation storage
is not permitted by the published default storage rule; no separate authorization
has been established. This is a concrete conflict with the
[domain history requirement](DOMAIN_MODEL.md), independent of whether a key can
be issued. Hence BLOCKED, not approved subject merely to obtaining a key.

The terms do not expressly adjudicate every possible derived statistic. No
permission for such derived retention is inferred, and replacing observations
with aggregates is not proposed. The provider contract and
[architecture](ARCHITECTURE.md) need no change.

### Next action and reopening criteria

Keep implementation on hold. If pursuing an exception through the
[official contact route](https://developer.bestbuy.com/contact-us), the exact
questions are:

1. Are new Products API keys currently issued to independent developers, and what
   developer registration/approval steps follow Best Buy account creation?
2. Will Best Buy expressly authorize this tracker to retain indefinitely and
   display timestamped SKU-level price/stock observations normalized from its API,
   including observations older than 72 hours, for history, comparisons, and
   future alerts? Which agreement overrides the storage restriction and covers
   this use, and what attribution/display obligations remain?

These questions are recorded for follow-up; no message was sent. Reopen only
after applicable written authorization and actual access are established.

**Next candidate to investigate: Dell US through an authorized catalog/feed
agreement.** Its [official Catalog API](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0)
requires a Premier page, procurement integration, and Dell-team activation, and
returns organization-specific pricing. Investigate eligibility, historical
retention rights, and whether an appropriate consumer-price feed exists first.
Dell is an investigation candidate, not an approved substitute; business-specific
prices must not be presented as public consumer prices. No scraping fallback.

### Verification boundary

Official sources above were opened on the check date; the login/account UI was
inspected without submission. Published policy and documented fields are
distinguished from unverified issuance and live data. No credentials, code,
dependencies, or provider interface changes are part of this review.

## Dell access feasibility — Task 15

Checked: 2026-09-28. Outcome: **BLOCKED for this project's first real provider
under the published access model and API terms**.

This resolves the earlier recommendation to investigate Dell. The user-requested
feasibility review takes precedence over Task 15's ingestion work in
[TASKS.md](TASKS.md); no adapter or ingestion integration is authorized here.

### Access and available official routes

- **Catalog Pull API:** public documentation, restricted customer data. Dell
  requires an established Premier page, procurement-system integration, and
  Dell Integration Team activation/configuration. The documented contact is
  `Premier.API@dell.com`. The catalog contains organization-specific pricing;
  annual credential maintenance is required. A developer-portal account alone
  does not satisfy these prerequisites.
  [Catalog API 2.0 overview](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0).
- **Catalog Push API:** the same customer catalog delivered to a customer webhook,
  with Premier/procurement onboarding and Dell activation. It supports OAuth2,
  API-key, or HTTP Basic protection for that webhook. It does not establish a
  public consumer feed and is unnecessary for our single-listing pull contract.
  [Catalog Push 2.0](https://developer.dell.com/apis/15788/versions/2.0.0).
- **Quote API:** Premier/Dell-team onboarding; retrieves a numbered, versioned,
  localized quote, with item/SKU and deal/tax/shipping information. This is a
  customer quote context, not evidence of an anonymous store-price service.
  The inspected v3 documentation publishes one request per second; do not apply
  that limit to Catalog. A v4 landing page exists, but its endpoint details were
  not exposed in this review.
  [Quote v3 documentation](https://developer.dell.com/apis/51341c61-7122-4610-acce-7cf920bfb4c7/versions/3.0.0/swagger.json),
  [v4 landing page](https://developer.dell.com/apis/51341c61-7122-4610-acce-7cf920bfb4c7/versions/4.0.0).
- **Other commerce/official access:** the public
  [API directory](https://developer.dell.com/apis) distinguishes documentation
  visibility from subscriptions and lists customer/commerce integrations. No
  reviewed official source established an unrestricted consumer-price API or
  an affiliate feed licensed for permanent history. This is a research boundary,
  not a claim that no negotiated feed can exist. Website visibility is not such
  a license; Dell's [AUP](https://www.dell.com/en-us/legal/lp/acceptable-use-policy)
  restricts high-volume automated access and unauthorized access. No scraping
  route is proposed.

**Credential feasibility:** an eligible Premier organization has a documented
onboarding route. Access for a new independent price-tracker developer has not
been demonstrated. No application, account registration, or credential request
was submitted; approval criteria beyond the published prerequisites are unknown.
Do not describe new issuance as either guaranteed or closed.

### Authentication and endpoints

Catalog 2.0 documents subscription `client_id`/`client_secret` in My Apps,
OAuth client credentials, and `Authorization: Bearer` authentication. Its token
section and cURL example name:

```text
POST https://dellidentity-corp.dell.com/di/proxy/l7/api/v3/oauth/token
grant_type=client_credentials
```

The same page's HTTP example instead uses
`https://apigtwb2c.us.dell.com/auth/oauth/v2/token`. This documentation mismatch
must be resolved with Dell for the assigned subscription; do not silently pick
an endpoint. The documented token lifetime is one hour; token reuse instructions
are not a catalog-data retention license.
[Catalog authentication reference](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0/getting-an-authorization-token-11431m0).

The overview and authentication example document this catalog route (the latter
uses `CustomerName` for the path placeholder):

```text
POST https://apigtwb2c.us.dell.com/PROD/CatalogAPI/Catalog/Search/{PartnerIdentifier}?ctry={country}&ccy={currency}
```

No authenticated request was made. Country, currency, and customer identifiers
must match Dell's provisioned context, not arbitrary public-store selections.
The overview labels 2.0 recommended but lists its availability as TBD and 1.0 as
default; production version entitlement therefore also requires confirmation.
[Catalog overview](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0).

### Data availability and semantic limits

- **Product/Listing identity:** Catalog v1 explicitly describes product, module,
  and SKU data. It does not establish permanent identifier stability or mapping
  to an exact public retail Listing URL. Configuration and customer context would
  need validation before accepting observations.
  [Catalog v1](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/1.0.0).
- **Current price:** pricing/configuration data is documented, but for the
  organization's catalog. Public consumer offer-price equivalence, freshness,
  and exact price-field semantics were not verified.
  [Catalog 2.0](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0).
- **Currency:** the catalog route explicitly accepts `ccy` and `ctry`; the
  overview links ISO currency/country references. An actual response's currency
  field and agreement with requested context remain unverified. Never infer USD
  solely from the API hostname.
- **List price/MSRP:** no explicit, verified MSRP or retailer-list-price mapping
  was established from the readable catalog documentation. These remain unknown,
  not aliases for a catalog price or discount basis.
- **Stock/availability:** no verified inventory field or reliable in-stock /
  out-of-stock mapping was established. Catalog membership, removal events, and
  quote existence do not by themselves establish stock.
- **Promotions:** Quote documents deal information, but consumer coupon,
  membership, cashback, and promotion eligibility semantics remain unverified.
  No Promotion or EffectivePrice behavior is selected.

The linked [Catalog endpoint page](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0/catalogsearch-329047e0)
rendered a documentation shell without endpoint/schema details in both web
extraction and the browser. The linked Push payload could not be retrieved.
An older mapping PDF surfaced in official-source search but returned 403 on
direct retrieval; it was not used as a verified current schema. These limitations
mean unknown fields are not asserted to be absent from Dell's actual API.

### Operational and historical-storage constraints

The [current API terms landing page](https://www.dell.com/en-us/lp/legal/api-terms-of-use)
links the [English API Terms of Use, APR2023](https://i.dell.com/sites/csdocuments/Legal_Docs/en/us/api-terms-of-use_en.pdf).
Relevant provisions:

- Section 4.1 limits the license to contemplated internal business purposes;
  licenses cannot be implied.
- Section 5.1 defines Application Data as data pulled or pushed through the API.
- Section 8 requires deletion of all Application Data on termination.
- Sections 5.2, 7, and 14.4 cover usage limits, confidentiality, and trade controls.
- Section 5.3(K) restricts availability/performance monitoring and benchmarking;
  its application to retail stock tracking needs Dell's clarification.

No numeric catalog quota, cache TTL, or explicit permanent-history exception was
found in the reviewed terms/catalog pages. Absence of a TTL is not permission.
Normalization does not establish an exemption from deletion. This conflicts with
our permanent observation requirement even for an internal deployment; public
display additionally exceeds the ordinary internal-use grant without permission.
These are the reasons for the BLOCKED project decision, not claims that all
internal purchasing analysis is prohibited.

**Other unresolved operational details:** per-account catalog rate/burst limits,
country coverage and entitlements, response freshness, and a self-service sandbox
or test credentials were not established from the reviewed Catalog pages. Public
examples are documentation, not proof of a usable sandbox. Confirm each during
any future authorized onboarding; do not invent quotas or test against production.

### Next action

Do not implement Dell under the default terms. Reopening requires Dell to confirm
eligibility and provide applicable authorization for retained observations,
including after API termination, and the intended history/display/comparison
use. Also request the provisioned version/schema, auth endpoint, country/currency
scope, quotas, and test access. The documented Premier contact above is a route
for these questions; no message was sent.

**Next investigation candidate: eBay Browse API**, which has an official
[listing lookup interface](https://developer.ebay.com/develop/api/buy).
This is a research candidate only: first assess production eligibility and
historical-retention rights, then seller/condition/variant identity suitability.
No feasibility approval or implementation selection is implied; no deeper eBay
research belongs to this task.

### Verification boundary

Official Dell pages and the linked English terms were checked on the date above;
retrieval limitations are recorded rather than hidden. No credentials or live
data were obtained. The decision does not depend on assuming missing fields or
denied credential issuance. Best Buy's earlier outcome remains unchanged.
Architecture, provider interfaces, domain code, and roadmap numbering are unchanged.

## eBay access feasibility — Task 16

Checked: 2026-09-28. Outcome: **BLOCKED under the published API license for
permanent historical observations**.

This resolves the preceding recommendation to investigate eBay. The requested
feasibility gate takes precedence over collection scheduling listed as Task 16
in [TASKS.md](TASKS.md); roadmap numbering is unchanged. No real provider is
approved. This is a project suitability decision under default terms, not a
claim that eBay has rejected an application or prohibits every historical use.

### Access: an application route, not guaranteed production approval

A Developers Program account and application keyset are required. Sandbox and
production have separate keys; a keyset includes App ID (Client ID), Dev ID,
and Cert ID (Client Secret). Creating production keys does not itself establish
permission for the proposed application.
[Getting started and application keys](https://developer.ebay.com/develop/guides/sell/get-started-with-ebay-apis).

The published Buy API production process includes eBay/developer accounts,
eBay Partner Network application and business-model assessment, approval,
a developer-support production-access request, application review, and applicable
agreements. Sandbox experimentation is available to developers; some checkout
methods require additional approval. Do not confuse sandbox access or
checkout-specific requirements with approval of a Browse price-history app.
[Buy API requirements](https://developer.ebay.com/api-docs/buy/buy-requirements.html).

**Independent developer feasibility:** there is a documented application route,
but this project's business model, production entitlement, and new credential
issuance remain unverified. No account was registered, approval requested, or
authenticated request executed. Public documentation is not proof of acceptance.

Browse uses an OAuth Application access token obtained with client credentials,
then `Authorization: Bearer`. Public listing reads do not require a shopper's
delegated sign-in. The token endpoint is
`POST https://api.ebay.com/identity/v1/oauth2/token`; the basic scope is
`https://api.ebay.com/oauth/api_scope`. Credentials must remain server-side.
[Browse overview](https://developer.ebay.com/develop/api/buy/browse_api),
[OAuth authorization](https://developer.ebay.com/develop/guides/sell/authorization).

Production-key activation also requires marketplace-account-deletion notification
compliance or an applicable exemption. The documented no-data-persistence opt-out
does not fit this tracker. Seller-linked data would need appropriate deletion
handling; do not assume price-only storage removes every obligation.
[Account-deletion requirements](https://developer.ebay.com/develop/guides/sell/marketplace-user-account-deletion).

### Relevant API and documented data

Browse is the smallest relevant API: search discovers public listings, while
`getItem` retrieves one exact listing's price and availability. A separate bulk
feed or catalog integration is unnecessary for this task's proposed scope.
[Browse API guide](https://developer.ebay.com/api-docs/buy/api-browse.html).

Production routes, documented but not executed:

```text
GET https://api.ebay.com/buy/browse/v1/item/{item_id}
GET https://api.ebay.com/buy/browse/v1/item/get_item_by_legacy_id?legacy_item_id={id}
GET https://api.ebay.com/buy/browse/v1/item_summary/search?q={query}
```

The current [Browse OpenAPI specification, v1.20.4](https://developer.ebay.com/develop/api/spec/browse_api.json)
documents these response capabilities:

- `itemId`: listing/variation identity, distinct from a catalog Product;
  `itemWebUrl`: listing URL. Preserve the complete REST item ID.
- `price.value` and `price.currency`: item price and currency.
- `marketingPrice.originalPrice`: pre-discount reference amount;
  `priceTreatment` supplies presentation context. This is not automatically MSRP.
- `estimatedAvailabilities`: availability status and estimated quantity where
  supplied; quantity may be thresholded, not exact inventory.
- `seller.userId`/`seller.username`, `listingMarketplaceId`: seller and market
  context. Username availability is restricted in some contexts; do not rely on
  a display name as permanent seller identity.
- `buyingOptions`, condition fields, and shipping options support scope checks.
- Discount amounts/percentages and `availableCoupons` can carry promotion data;
  presence is conditional, not a universal offer or eligibility guarantee.

The guide recommends checking `itemEndDate` together with availability; an ended
listing can still be returned. Its `COMPACT` response supports refreshing price
and availability, not permission to archive them indefinitely.
[Availability and refresh guidance](https://developer.ebay.com/api-docs/buy/api-browse.html).

### Quotas and historical retention

Published default Browse limits are **5,000 calls/day for methods other than
`getItems`**, with a separate 5,000/day limit for `getItems`. Higher limits require
an Application Growth Check. Buy APIs require an additional license. Actual
assigned quotas and burst limits were not verified without an account.
[Official API call limits](https://developer.ebay.com/develop/get-started/api-call-limits).

The currently served [API License Agreement](https://developer.ebay.com/join/api-license-agreement)
is dated September 3, 2025. The relevant distinctions are:

- **Raw responses:** the eBay Content definition covers retrieved data, not just
  complete documents. Section 3.1 permits necessary intermediate copies for
  authorized activity, with deletion when unnecessary.
- **Caching/display:** section 8.1(b) requires removing content no longer publicly
  available. Section 8.1(c) limits displayed listing-information age to six hours
  (other content: 24 hours). This is a display-freshness rule, not a blanket
  six-hour storage allowance.
- **Normalized history:** no explicit historical-observation exemption was found.
  Section 16.3 requires destruction of copies or material containing eBay Content
  within ten days after termination. Section 8.1(d) separately restricts certain
  derived analytics without written permission; it does not expressly resolve
  this exact per-listing chart use case.

**Project determination:** converting API prices to minor units and adding a
timestamp does not establish independent data rights. Permanent retention
conflicts with default deletion obligations. Whether timestamped charts satisfy
display/analytics rules also needs clarification; no permission is inferred.

### Product-model fit: hypothetical scope only

If separately authorized, the following deliberately narrow scope would be
technically plausible; it is **not implementation approval**:

- Track one manually selected eBay Listing by complete item/variation ID, on
  one explicit marketplace, with verified seller and new-condition context.
  Require fixed-price eligibility and reject any auction buying option, including
  auctions that also offer Buy It Now. Exclude used/refurbished items.
- Map the item amount to OfferPrice with its returned currency. Leave MSRP,
  retailer list price, and sale price absent until their exact semantics are
  explicitly established; a reference price is not manufacturer evidence.
- Map supported availability to stock state; absent or insufficient evidence
  stays unknown. Request/access failures remain errors, not stock-only success.
- Exclude shipping, taxes, coupons, negotiated offers, and member discounts from
  the observed item price. No EffectivePrice, promotion evaluation, bidding,
  checkout, or cross-seller best-price aggregation.
- Keep Product retailer-independent and Listing specific to the source item.
  Multiple sellers can therefore correspond to multiple Listings for one
  manually identified Product, without automatic matching.

The current catalog types defer marketplace/seller modeling and do not store
first-class condition or marketplace fields. Before any eventual adapter,
review how the restricted seller/marketplace/variation context will be verified
and preserved; do not silently conflate eBay with the selling merchant. This is
a documented modeling boundary, not an interface change. The existing Provider
contract can return normalized PriceObservations without database access;
historical-use authorization remains the prerequisite.

### Next action and verification boundary

Keep eBay implementation on hold. Reopening requires production approval for
this business model and applicable written authorization answering:

1. May this application retain and display indefinitely timestamped item-level
   price/stock observations, including after listings cease to be public and
   after API agreement termination? Which terms override the deletion rules?
2. How do freshness and derived-analytics restrictions apply to explicitly dated
   historical charts? What refresh, attribution, separation from other sources,
   and seller-account-deletion duties remain?

No support request was sent. **Next investigation direction:** a cooperating
merchant's official API/feed with explicit contractual permission for historical
retention and charts. Establish those rights before researching adapter details;
no replacement provider is selected or approved here.

Official sources were checked on the date above, including the current machine-
readable Browse schema. Documented capabilities are not live-data verification.
The decision does not depend on assuming credentials are unavailable. Prior
Best Buy/Dell outcomes, [domain semantics](DOMAIN_MODEL.md),
[architecture](ARCHITECTURE.md), and provider interfaces remain unchanged.
