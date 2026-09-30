# Provider access feasibility

Latest review: [France domain fit, Task 18](#france-fuel-feed-domain-fit--task-18).
The earlier individual feasibility decisions are retained below for traceability.

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

## Provider candidate sweep — Task 17

Checked: 2026-09-29. Nine candidates across retail, manufacturers, marketplaces,
commerce platforms, and open price datasets. Three existing candidates are
rechecked baselines; six are newly compared here. Candidates were chosen for
documented price interfaces or explicit data-reuse grants, not popularity.

This requested research task supersedes current-price query work for this turn.
Roadmap IDs are unchanged. The provider-selection gate in [TASKS.md](TASKS.md)
now records this sweep; no adapter is approved or implemented.

### Classification and evidence rules

- **PROMISING:** practical documented access and affirmative reuse rights support
  historical retention. Worth a focused feasibility task; not implementation
  approval. Remaining data/model constraints must be stated.
- **BLOCKED:** published restrictions conflict with this project's intended use
  under the default terms. A negotiated exception would be a new decision.
- **NEEDS CLARIFICATION:** material rights, source access, or suitability remain
  unresolved. Public responses and missing cache limits are not permission.

Raw-response storage, temporary caches, and normalized historical observations
are separate uses. Converting a price to minor units does not by itself remove
source-license restrictions. Conversely, an affirmative open-data grant to copy,
transform, and reuse data can support history without a special cache exception.
All licenses still require compliance with their conditions.

The summaries below concern published capabilities. No accounts, credentials,
merchant agreements, live authenticated requests, or support submissions were
made. An unknown quota means unverified, not unlimited. Stable source IDs identify
records within their source context; they are not automatic Product matches or
guarantees against future identifier changes.

### 1. Best Buy Products API — BLOCKED

- **Access/data:** developer registration and API key; SKU-based lookup, current
  `salePrice`, `regularPrice`, and online availability are documented. New key
  issuance remains unverified. US pricing context would need an explicit USD
  mapping; do not infer currency from a number alone.
  [Products reference](https://bestbuyapis.github.io/api-documentation/).
- **Limits/retention:** 5 calls/second and 50,000/day per key. Storage/caching is
  limited to 72 hours for faster display. This covers Content, not only raw JSON;
  no historical-observation exception is established.
  [API terms](https://developer.bestbuy.com/legal).
- **Practicality/restrictions:** narrow SKU fixtures would be easy technically,
  but default retention conflicts with history. The
  [Task 14 decision](#best-buy-access-feasibility--task-14) remains in force.

### 2. Dell Premier Catalog API — BLOCKED

- **Access/data:** Premier customer context, procurement integration, and Dell
  activation; OAuth credentials. Catalog/configuration identity and negotiated
  pricing are documented, with country/currency context. Public-store identity,
  exact response currency, and reliable stock mapping remain unverified.
  [Catalog overview](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0).
- **Limits:** no numeric Catalog quota or self-service sandbox was established;
  do not borrow Quote API limits.
- **Retention/restrictions:** internal-business licensing and deletion of API
  Application Data on termination conflict with the intended public, retained
  history. Normalization is not an established exemption.
  [API terms, sections 4, 5, and 8](https://i.dell.com/sites/csdocuments/Legal_Docs/en/us/api-terms-of-use_en.pdf).
- **Practicality:** customer-specific onboarding/data make this a poor independent
  developer starting point. [Task 15](#dell-access-feasibility--task-15) records
  the schema/auth documentation limitations; access denial is not assumed.

### 3. eBay Browse API — BLOCKED

- **Access/data:** developer keys, OAuth application token, and separate Buy
  production approval; exact item/variation IDs, price/currency, seller context,
  and estimated availability are documented.
  [Requirements](https://developer.ebay.com/api-docs/buy/buy-requirements.html),
  [Browse schema](https://developer.ebay.com/develop/api/spec/browse_api.json).
- **Limits:** default 5,000/day for Browse methods except `getItems`, which has
  its own 5,000/day allocation; account entitlement remains unverified.
  [Call limits](https://developer.ebay.com/develop/get-started/api-call-limits).
- **Retention/restrictions:** content-removal and post-termination destruction
  duties conflict with permanent history; dated charts also raise unresolved
  display/analytics questions. No normalized-price exemption was established.
  [API license, sections 8 and 16](https://developer.ebay.com/join/api-license-agreement).
- **Practicality:** sandbox/fixtures and fixed-price, new-condition scope help
  testing, but cannot resolve policy. Seller, variation, and shipping context
  remain important. [Task 16](#ebay-access-feasibility--task-16) has the details.

### 4. Amazon Creators API — BLOCKED

- **Access:** Associates enrollment, API registration/credentials, and qualifying
  sales. Current eligibility documentation specifies ten qualifying sales in
  the preceding 30 days; this is not an unconditional hobby-developer key.
  [Prerequisites](https://affiliate-program.amazon.com/creatorsapi/docs/),
  [Eligibility errors](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/troubleshooting/error-codes-and-messages).
- **Data:** ASIN lookup through GetItems; OffersV2 supplies price/currency,
  availability, seller, and reference-price context. ASIN alone does not fix a
  seller/condition offer; saving basis is not automatically MSRP.
  [OffersV2](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/api-reference/resources/offersV2).
- **Limits/testing:** documented initial maximum 1 request/second and 8,640/day
  for 30 days, then sales-linked allocations/access. Use examples/fixtures only
  until eligible credentials exist.
  [API rates](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/concepts/api-rates).
- **Retention/restrictions:** Participation Requirements 3(y) prohibit price
  tracking/alerts unless Amazon agrees otherwise. The license permits non-image
  advertising-content caching up to 24 hours with refresh; indefinite ASIN
  storage is not permission to archive its prices. Normalized history is not an
  established exception. This explicit use restriction decides the outcome.
  [Associates policies](https://affiliate-program.amazon.com/help/operating/policies).

### 5. Shopify Storefront API across merchant stores — BLOCKED

- **Access/data:** Storefront supports tokenless and token-based access, subject
  to permissions. Product/variant IDs, contextual price/currency, compare-at
  price, and sale availability are documented. Technical public access does not
  establish permission to harvest arbitrary shops.
  [Storefront reference](https://shopify.dev/docs/api/storefront/2026-04),
  [Product query](https://shopify.dev/docs/api/storefront/latest/queries/product),
  [Product fields](https://shopify.dev/docs/api/storefront/latest/objects/Product).
- **Limits:** buyer traffic has no fixed request-per-minute limit; automated
  traffic is limited. Tokenless query complexity is capped at 1,000. Do not apply
  the buyer exemption to this collector. Fixtures are straightforward once an
  authorized store context exists.
  [Storefront limits](https://shopify.dev/docs/api/storefront/2026-04#rate-limits).
- **Retention/restrictions:** API terms section 2.3.14 restrict systematic
  collection/product indexes; section 6 limits Merchant Data use to authorized
  services and requires deletion within 30 days of specified triggers, including
  uninstall or enforceable deletion requests. No normalized-history carve-out
  was established. A merchant's technical token does not override platform terms.
  [API license](https://www.shopify.com/legal/api-terms).
- **Boundary:** this classification concerns the proposed cross-store tracker,
  not a claim that every merchant-authorized internal analytics app is forbidden.

### 6. One consenting WooCommerce merchant, Store API — NEEDS CLARIFICATION

- **Access/data:** official public product endpoints expose published product
  IDs, permalinks, price/regular/sale fields, currency/minor-unit information,
  and stock indicators. Storefront reads do not require merchant API keys.
  [Store API](https://developer.woocommerce.com/docs/apis/store-api/),
  [Products](https://developer.woocommerce.com/docs/apis/store-api/resources-endpoints/products).
- **Limits/testing:** the built-in optional limiter's documented 25 requests per
  ten seconds applies to POST, not a blanket GET quota. A merchant/host can impose
  additional controls. An owned test store permits repeatable fixtures, but is
  not proof of access to a production merchant.
  [Rate limiting](https://developer.woocommerce.com/docs/apis/store-api/rate-limiting).
- **Retention/restrictions:** these technical docs do not grant a license to
  every merchant's catalog. No concrete merchant or history agreement is in hand.
  Raw caching and retained normalized observations both require that source's
  policy to be settled. Open-source server software is not a catalog-data license.
- **Next gate:** identify one merchant willing to authorize bounded reads and
  indefinite price-history reuse, including after access ends. Define variants,
  currency, tax context, and attribution. Until then this is an access route,
  not a PROMISING licensed source or an approved retailer.

### 7. Open Food Facts Open Prices — NEEDS CLARIFICATION

- **Access/data:** public price reads and daily exports; authenticated operations
  use an Open Food Facts account/Bearer token, with a pre-production environment.
  Records associate product codes, location, price/currency, and observation
  dates. No stock field or continuously refreshed offer guarantee was established.
  [API guide](https://openfoodfacts.github.io/open-prices/guides/API/),
  [Price query reference](https://openfoodfacts.github.io/documentation/docs/Open-prices/prices/prices_list/),
  [Read access overview](https://openfoodfacts.github.io/documentation/docs/).
- **Limits:** no numeric Open Prices read quota was found in those guides; do not
  substitute the separate Product Opener API limits. Daily dumps reduce API load.
- **Retention:** the project explicitly publishes data under ODbL and provides
  reusable exports. Retained observations are compatible in principle with that
  grant, subject to attribution/share-alike and database-combination obligations;
  this is affirmative reuse evidence, not an inference from a missing cache TTL.
  [Data and license](https://openfoodfacts.github.io/open-prices/guides/data/).
- **Unresolved suitability:** receipt/shelf observations can be old and are not
  retailer-current offers. A product/location pair is not necessarily a product
  page Listing. Clarify freshness, source dates versus collection timestamps, and
  the intended ODbL distribution of our combined database before selection.
  Do not relabel a historical receipt as today's price.
  [Source model](https://openfoodfacts.github.io/open-prices/topics/core/).

### 8. France government fuel-price open data — PROMISING

- **Access/data:** free public ZIP/XML downloads without developer approval.
  Station ID plus fuel ID identifies a station/fuel record; price is explicitly
  EUR with source-update time. The instantaneous feed includes outages and
  closures. Names/brands of stations are excluded from the source feed, so do
  not invent merchant names. No numeric request quota is published on this page.
  [Official feed and field dictionary](https://www.prix-carburants.gouv.fr/rubrique/opendata/).
- **Freshness/testing:** the source feed updates every ten minutes; the improved
  government mirror harvests every fifteen minutes. Source price-update times
  still matter. Prefer the original feed; published archives support fixtures.
  [Government dataset metadata](https://www.data.gouv.fr/datasets/prix-des-carburants-en-france-flux-instantane-v2-amelioree).
- **Retention:** the dataset identifies Open Licence 2.0. Its affirmative grant
  covers copying, transformation/derived information, publication, and commercial
  reuse for unlimited duration. Attribute source and update date; do not imply
  endorsement or misrepresent data. Raw snapshots and normalized histories are
  therefore compatible, not merely temporarily cacheable.
  [Official license text](https://github.com/etalab/licence-ouverte/blob/master/LO.md).
- **Model/practicality gate:** fuel is a unit-priced commodity, not a packaged
  retail SKU. Validate exact decimal precision, unit basis, source detail URL,
  station identity continuity, and outage-to-stock mapping before implementation.
  Missing outage reports must not automatically establish in-stock status.
  EUR is not yet supported by our domain code. This is a viable research target,
  not a drop-in adapter or an approved change of product scope.

### 9. Italy MIMIT fuel-price open data — PROMISING

- **Access/data:** public daily station and price files, no developer approval.
  They report prices effective at 08:00 on the previous day, not a live quote.
  Since February 10, 2026 the delimiter is `|`, despite the CSV designation.
  No numeric download quota is published on the dataset page.
  [Official dataset](https://www.mimit.gov.it/it/open-data/elenco-dataset/carburanti-prezzi-praticati-e-anagrafica-degli-impianti).
- **Identity/stock:** `idimpianto`, fuel description, and self-service/served flag
  define the price context. Amounts are EUR to three decimal places per litre
  (methane: kilogram), with a communication timestamp. Station metadata includes
  operator and name. No stock field is documented; preserve unknown.
  [Current field specification](https://www.mimit.gov.it/images/stories/documenti/Metadati_prezzi_carburanti_20260128.pdf).
- **Retention:** MIMIT explicitly licenses the dataset under IODL 2.0. Official
  government explanations of that license permit reproduction, extraction,
  reuse, and derivative works with attribution and non-misrepresentation duties.
  These are compatible with retained histories, not only temporary caching.
  [Government IODL explanation](https://www.dati.lombardia.it/legale/normativa),
  [Government reproduction of reuse conditions](https://porfesr.regione.campania.it/it/menu-servizio/privacy-e-note-legali/note-legali-qmbc?page=1).
- **Practicality/restrictions:** simple files make fixtures practical; daily lag,
  experimental publication status, and unit-price precision make it weaker than
  France for current-price tracking. The full license link on MIMIT returned 403
  in this research environment; the licensing assertion and official explanations
  above were readable. Confirm the complete applicable text in focused review.
  Treat records as dated reported prices, never assured live availability.

### Ranking: PROMISING candidates only

Historical Task 17 ranking: France's focused review below subsequently
[rejected it as the first provider](#france-fuel-feed-domain-fit--task-18).
The ranking is retained as research history, not current implementation approval.

1. **France government fuel feed.** Best documented combination of independent
   access, recent reported prices, stock-out evidence, source IDs, reusable
   archives, and explicit unlimited-duration reuse. First choice for the next
   feasibility task, provided fuel tracking is an acceptable initial context.
2. **Italy MIMIT fuel files.** Accessible and reuse-compatible, with easy fixtures,
   but slower snapshots and no documented stock state. Appropriate only if daily
   reported-price history is useful; not a substitute for live retailer quotes.

These ranks assess sources worth validating, not ready-to-implement adapters.
No conventional retailer or marketplace in this sweep clears the historical-use
gate. If the first provider must be a packaged-goods store, pursue the named
WooCommerce-merchant permission gate instead; it remains unranked until granted.

### Next feasibility task and model guardrails

Validate the France feed narrowly: one station and one fuel grade, official
source URL/identity, dated EUR offer price, supported outage facts, and attribution.
Check a representative current record and archive, decimal scale, timezone,
source update versus retrieval time, and a conservative download interval.

The current Money type supports only USD/JPY integer minor units. Fuel unit
prices may require sub-cent precision; Italy explicitly documents three decimals.
Do not round away source evidence, redefine EUR's exponent, or turn a calculated
ten-litre total into an observed offer. Determine whether a small separately
reviewed unit-price representation is justified or whether this category should
be rejected. No currency, domain, persistence, or provider changes are authorized
by this research. A government feed is the Provider source, not the Retailer;
the Retailer must retain the actual station/selling context.

The narrower retention grant is not the only gate: Listing URL/context,
observation-time precision, refresh behavior, and trustworthy price semantics
must also fit [DOMAIN_MODEL.md](DOMAIN_MODEL.md). Missing fields stay missing;
reference amounts are not MSRP; operational failures are not stock observations.
The [architecture](ARCHITECTURE.md) and existing contract remain unchanged.

### Verification boundary

Important findings use current official documentation, dataset metadata, and
licenses checked for this sweep. Earlier detailed decisions remain historical
records. No private permissions are assumed. The old Etalab license landing URL
redirected to the data portal; the official Etalab license repository supplies
the text, and the government dataset explicitly identifies version 2.0. Italy's
full-license retrieval limitation is recorded above. Numeric limits not found
are left unknown. Source capability is distinguished from actual credential
issuance, live response validation, and implementation approval.

## France fuel feed domain fit — Task 18

Checked: 2026-09-29. Outcome: **REJECTED FOR THIS PROJECT as the first real
provider**.

This supersedes Task 17's provisional France recommendation. Its reuse license
remains suitable, but exact unit pricing requires a broader price representation
before ordinary retail integration. That is not justified merely to obtain a
first provider. This rejects the current selection, not the possibility of a
separately scoped commodity-price feature later. No domain change is approved.

### Evidence checked

- The [official feed dictionary](https://www.prix-carburants.gouv.fr/rubrique/opendata/)
  defines station `pdv/@id`, fuel `prix/@id`, EUR `valeur`, and `maj` as the last
  price update. It also describes address/coordinates, outages, closures, and
  archives. Station names and brands are excluded from this feed.
- The public [instantaneous ZIP/XML feed](https://donnees.roulez-eco.fr/opendata/instantane_ruptures)
  was fetched and parsed in memory on the check date, without writing a fixture
  or application code. One returned station was `89100001`; its Gazole record
  contained the following attributes, copied as strings:

  ```xml
  <prix nom="Gazole" id="1" maj="2026-09-28 11:45:27" valeur="2.429" />
  ```

- An [official station detail page](https://www.prix-carburants.gouv.fr/station/34600001)
  explicitly labels prices EUR per litre and displays three-decimal amounts for
  several grades. It also shows that one station page contains multiple fuels.
  This page was read for feasibility, not proposed as a scraping source.

### Domain mapping: plausible relationships, incomplete price semantics

**Product:** a precisely defined fuel grade, independent of station, could be a
Product. The source's grade code is useful evidence, but a broad label such as
Gazole is not proof that branded/additivated fuels are identical products. Any
mapping would remain manual and narrowly defined; no matching by fuel label
alone across all retailers. Product must not acquire a station or coordinates.

**Retailer:** the individual station/store context is the natural initial
Retailer, not the government publisher and not every station sharing a brand.
The existing Retailer has ID and optional Name, so a source station ID can identify
the context without inventing a name. Operator, brand, and location are different
concepts; the feed does not establish a legal-operator identity. Its address and
coordinates can help check continuity, but the current type has no location or
operator-history fields. A geographic search feature is unnecessary for tracking
one explicit station.

**Listing:** `(station ID, fuel ID)` is a plausible source identity. One station
would have separate Listings for separate grades, each linked to its Product and
station Retailer. A station detail URL plus explicit fuel reference can select
the grade on a multi-product page. Existing Listing fields and the catalog's
`(retailer_id, url, retailer_product_id)` uniqueness constraint can express this;
there is no need to invent one merchant per fuel or fabricate URL fragments.
The source page is government-hosted rather than a merchant checkout, so this
would need an explicit accepted source-context interpretation, not a claim that
the government sells fuel.

**PriceObservation:** an immutable snapshot of one station/grade's reported unit
price and supported stock evidence is conceptually appropriate. It is an offer
unit rate, not MSRP, retailer list price, sale price, or a purchased-volume total.
An explicit active shortage can support out-of-stock for that grade; missing
shortage evidence does not prove in-stock. Closure and permanent non-distribution
must not be mistaken for a fresh zero price. The relationship model is therefore
not the decisive problem: the current monetary fields cannot hold the rate exactly.

These are proposed mappings, not newly accepted domain semantics. They were
compared with [DOMAIN_MODEL.md](DOMAIN_MODEL.md),
[catalog types](../backend/domain/catalog.go), and
[observation types](../backend/domain/observation.go).

### Money: exact incompatibility, even after hypothetical EUR support

The observed `2.429` EUR/L requires thousandths of a euro per litre. Its exact
representation is `2429 / 1000 EUR per litre`, or `242.9` euro cents per litre.
No floating-point arithmetic is needed to establish this incompatibility.
Euro cents are hundredths of a euro.
[ECB currency explanation](https://www.ecb.europa.eu/pub/pdf/other/eurobren.pdf).

[Money](../backend/domain/money.go) currently supports only USD and JPY and stores
an `int64` number of currency minor units. EUR is rejected today. Adding EUR with
its correct exponent of two would still not represent this price exactly.
Three decimals were verified in actual source data; this review does not claim
the field dictionary guarantees an exhaustive maximum precision for every archive.

Rounding to cents loses an observed fact. Using an exponent of three for EUR
misstates its currency scale. Multiplying by ten litres creates a derived total,
not the source's observed one-litre rate. Hiding precision in Source or emitting
stock-only observations would not satisfy the price-tracking goal.

Supporting this category honestly would require a separately reviewed exact
unit-price concept carrying currency, decimal/rational scale, and quantity unit,
distinct from payable Money amounts, plus corresponding observation/storage/read
semantics. That broader requirement is recorded only to explain the rejection;
it is not a proposed implementation task or permission to alter Money.

### Timestamps: update time is not a fresh collection time

`prix/@maj` is a last-update timestamp, not the time our collector retrieved the
file. In the live sample above it has seconds but neither an offset nor a `Z`.
The feed dictionary documents the same offset-free format. Outage and closure
events have their own start/end fields; their times need not match price updates.
[Timestamp dictionary](https://www.prix-carburants.gouv.fr/rubrique/opendata/).

The [government mirror metadata](https://www.data.economie.gouv.fr/explore/dataset/prix-des-carburants-en-france-flux-instantane-v2/?q=recordid%3Aecf07d8c81a8614a80b2ae14d447f3b6e2314f66)
identifies `Europe/Paris`. This supports a local-time interpretation for that
dataset, not an assertion that the original XML is UTC. The reviewed original
dictionary does not specify a timezone or daylight-saving ambiguity policy;
mirror metadata alone does not settle every original historical timestamp.

**Safe direct mapping to ObservedAt: not established.** The constructor converts
an already meaningful `time.Time` to UTC; it cannot recover a missing timezone
or distinguish an ambiguous repeated local time. Do not append `Z` or silently
use the backend machine's timezone. A source-update mapping would require a
confirmed timezone and explicit handling of ambiguous/invalid local times.

A retrieval timestamp could accurately describe observing a feed snapshot, but
it is not a fresh retailer price update. Using it alone would conceal source age.
The current observation has one ObservedAt and no typed source-update/event-time
fields. Combining an old price with a newer shortage would need an explicit
snapshot/provenance policy before presenting them as one coherent observation.
No such policy or new fields are introduced here.

### History: license fits; identity and refresh have limits

The [government dataset](https://www.data.gouv.fr/datasets/prix-des-carburants-en-france-flux-instantane-v2-amelioree)
identifies Open Licence 2.0. The [official license](https://github.com/etalab/licence-ouverte/blob/master/LO.md)
permits copying, transformation, publication, and commercial reuse for unlimited
duration, with attribution/source-update information and no misleading endorsement.
Retaining raw snapshots and normalized history is compatible with those rights;
there is no need to infer a normalized-data exception to a temporary cache limit.

Documented station/fuel IDs provide a sensible continuity key for repeated
records, but no reviewed official source promises permanent non-reuse or explains
all operator changes/relocations. Historical station IDs must not be treated as
guaranteed permanent legal-seller IDs. Current identity is usable for investigation;
cross-year continuity was not validated in this task. Address/context changes
would require review before joining histories automatically.

The government metadata documents ten-minute source refresh and fifteen-minute
mirror harvesting. This cadence permits repeated snapshots; it does not mean
every station declares a new price at that cadence. The live sample included an
E85 price last updated on August 25 alongside September updates. Later successful
retrievals can confirm what the feed still reports, but must not refresh the
underlying price's update time. Exact persistence retries would still use the
existing ingestion identity; an unchanged source update is not a new price-change
event. The published archives are useful history evidence, not proof of immutable
records or station identity guarantees.

### Decision and scope boundary

Reject this first-provider choice because exact sub-cent unit rates require
price-domain and downstream representation work unrelated to the planned first
ordinary retail adapter. EUR alone is not the fix. Source-time semantics and
station/product continuity add further validation work. The license and basic
relationships are viable, but they do not outweigh this mismatch with the current
project scope. This is **REJECTED**, rather than NEEDS DOMAIN CHANGE as a request
to expand the project for fuel.

The earlier sweep ranking remains a dated research result; it does not authorize
France implementation. Do not automatically promote Italy: its documented
three-decimal unit rates have the same precision concern. Resume investigation
of a concrete ordinary-goods source with explicit retained-history permission,
such as the still-unresolved consenting-merchant route. No replacement is approved.

Only this decision document changes. The [architecture](ARCHITECTURE.md), Go
types, EUR support, dependencies, interfaces, and roadmap remain unchanged.

## Task 25: Real provider selection and implementation gate

Reviewed September 30, 2026. **Outcome: BLOCKED. Selected provider: none.**
This is a project implementation gate, not a claim that every candidate prohibits
historical tracking. Four third-party sources were checked because earlier
retailer decisions did not clear access/retention requirements, and the fuel
source failed domain fit. Public technical documentation alone does not establish
data-use rights. No account was registered, subscription purchased, or live
collection attempted during this review.

### Price API (metoda): NEEDS CLARIFICATION

The [official getting-started guide](https://readme.priceapi.com/docs/make-your-first-request)
offers trial registration and source/country-specific product and offer lookup,
including source identifiers rather than mandatory keyword matching. Its
[workflow](https://readme.priceapi.com/docs/basic-workflow) is asynchronous.
The [official terms](https://www.priceapi.com/legal/terms), rechecked September 30,
2026 through the official page's indexed text, retain the July 27, 2015 version
date. **Rights of Use, paragraph (2)** grants:

> a right of use, which is unlimited by time and non-exclusive, to use the contents for his own purposes

This covers content the user saves on or prints from the provider's Internet
site; nonpayment permits revocation. Paragraph (1) instead limits online
retrieval/display to the contract term and restricts modification/publication,
subject to express permission or site functionality. API JSON/CSV delivery is
described in §2.1; business eligibility and acceptance remain conditions
(§3.3–3.4).

- **Private/internal retention:** affirmative time-unlimited own-use permission
  exists for covered saved content; the earlier assessment omitted it.
- **Public display/redistribution:** own-use permission is not that license;
  §2.2 requires prior text-form approval for resale/third-party purposes.
- **Termination/nonpayment:** covered saved-content rights have no stated time
  limit, unlike online access. Ordinary cancellation is not equated here with
  nonpayment, which expressly permits revocation. This is not an irrevocable grant.
- **Normalized API history:** the terms do not expressly identify normalized
  historical observations as covered saved content. Confirm API export coverage
  and whether normalization is permitted despite paragraph (1)'s restrictions.

**NEEDS CLARIFICATION, not approved:** the missing evidence is now the grant's
application to our exact normalized-history use, rather than absence of any
retention grant. Ask whether timestamped price/currency/stock records derived
from API responses qualify, may be retained after an ordinary paid-up
cancellation, and may be displayed privately versus publicly. Confirm this
project's business eligibility and applicable contract. Currency and exact
seller/listing continuity still require source-specific validation.

**Revised comparison:** Price API now has stronger affirmative retention evidence
than Rainforest or Keepa's reviewed material, and avoids PriceCharting's explicit
subscription-end purge rule for covered content. It becomes the first policy
clarification follow-up; Keepa remains the strongest technical follow-up. This
is a research priority, not an approval or finding that normalized API history
is already licensed. No candidate currently clears the complete gate.

### PriceCharting: BLOCKED under standard published API terms

The [API documentation](https://www.pricecharting.com/api-documentation) requires
a subscription token, provides product IDs and condition-specific price values,
uses integer pennies for API prices, and specifies USD for CSV prices. It limits
API calls to one per second. Price values are not automatically executable
retailer offers or stock evidence; its separate marketplace interfaces need
separate identity/availability review.

The same documentation permits caching/server storage but requires purging all
API/CSV data when the subscription ends. Sharing through applications used by
others requires express permission and a commercial license. The
[site terms](https://www.pricecharting.com/page/terms-of-service) also distinguish
internal use from applications accessible to third parties. No normalized-history
exception was established. Temporary authorized caching therefore does not meet
this project's durable history requirement. A negotiated license could change
the decision; a normal subscription alone does not clear it.

### Rainforest API (Traject Data): NEEDS CLARIFICATION

The [official product request documentation](https://docs.trajectdata.com/rainforestapi/product-data-api/parameters/product)
documents an API key and lookup by Amazon domain plus ASIN, or product URL.
The [product site](https://trajectdata.com/ecommerce/rainforest-api/) advertises
commerce data access; this is a third-party API, not an Amazon-issued data license.
Its [FAQ](https://trajectdata.com/faqs/) explains that requests acquire live data
and no sandbox is offered. Exact plan quota and offer/currency/stock mapping would
still need confirmation before implementation.

The site's [current terms link](https://trajectdata.com/traject-data-terms-of-service/)
serves ScraperAPI Terms of Use effective November 18, 2025. It provides a limited
service license (§2.1), requires prior consent for applications interacting with
the service (§2.5), reserves proprietary rights (§8.2), and assigns responsibility
for third-party terms (§10). An enterprise agreement can override conflicts.
The relationship between these general clauses and an API subscription's data
rights is unresolved. No explicit permanent normalized-observation grant was
established. Ask which API-specific agreement governs, including retention after
termination and portfolio display. Do not assume paying an intermediary grants
rights over all upstream data or removes upstream restrictions.

### Keepa: NEEDS CLARIFICATION; strongest technical follow-up

The [official API overview](https://keepa.com/api-docs/) documents subscription
access, an account API key, and `/product` requests by marketplace and ASIN.
[Plans and tokens](https://keepa.com/api-docs/plans-tokens.html) describe monthly
billing and plan-specific token refill rates; unused tokens expire after an hour.
This is a documented purchase path, not verified credential issuance for this
project.

The [offer specification](https://keepa.com/api-docs/offer-object.html) provides
an offer ID stable across requests within a product, seller and condition,
locale minor-unit prices, update time, and optional stock history. Missing prices
use sentinels, not zero. Offers may be stale; stock is best-effort. An ASIN alone
must not merge different sellers, conditions, or fulfillment contexts into one
Listing. These fields make a narrow implementation technically plausible.

However, the reviewed technical pages do not settle permanent normalized-history
storage or display rights. The [public terms entry point](https://keepa.com/#!terms)
did not expose readable agreement text through this review's web retrieval.
Unofficial reposted terms were not used as authority. This is an evidence gap,
**not a finding that Keepa prohibits retention**. Access to historical data is
not itself permission to retain or redistribute it indefinitely. Obtain the
current applicable API agreement and written clarification before approval.

### Required evidence and narrow implementation boundary

Next action: resolve Price API's saved-content/API-normalization questions above
first. Keepa remains an alternative requiring its applicable API agreement and
an answer to this exact question:
may an independent developer retain timestamped item/seller identity, currency,
observed price and stock indefinitely as normalized PriceObservations, including
after subscription termination, and display those retained observations in this
project? Ask separately about private use and any future portfolio/public display,
required deletion/refresh, attribution, and upstream restrictions. This review
does not send that request or presume a favorable answer. A named consenting
merchant granting these rights remains an alternative; none is established here.

Raw-response caching, normalized historical storage, and redistribution are
different permissions. Discarding raw JSON does not automatically exempt the
retained facts from contractual restrictions. Conversely, missing explicit
permission here is not proof of a prohibition. No candidate clears all required
evidence, so implementation stops at documentation as requested.

If a future review clears Keepa, the proposed first scope is one explicitly
identified new-condition offer in one supported marketplace/currency, with its
seller/context and source update time preserved. Offer availability must be
validated; unknown stock remains unknown. Exclude cross-seller best-offer
switching, used/refurbished items, inferred MSRP/list/sale semantics, membership
prices, coupons, shipping-inclusive EffectivePrice, and automatic matching.
This is a conditional scope proposal, not an approved adapter design or interface
change. Existing Provider, ingestion, immutable-history, and manual/scheduled
collection code remain unchanged. No live success is claimed.
