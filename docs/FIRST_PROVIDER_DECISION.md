# First provider decision

**Superseded implementation status:** the [Task 14 feasibility review](PROVIDER_RESEARCH.md)
records Best Buy as **BLOCKED**. Its decision overrides the conditional adapter
handoff below; do not begin implementation on the basis of this Task 13 document.

Research date: 2026-09-28. Task 13 is documentation only. Read alongside
[the roadmap](TASKS.md), [architecture](ARCHITECTURE.md), and
[domain model](DOMAIN_MODEL.md).

## Decision and production-readiness status

Select **Best Buy US Products API as the preferred, conditional Task 14 target**.
It provides the clearest documented single-SKU interface and price/availability
semantics of the three candidates. This is an implementation-target selection,
not confirmation that production price tracking is authorized.

**No candidate is cleared for production history collection by this research.**
Best Buy's ordinary API terms limit content storage to 72 hours, conflicting with
our immutable price history. Before live collection or Task 15 persistence, obtain
written permission covering retained observations, historical display, future
alerts, and intended comparison use. Confirm actual API access as well. An API key
alone does not resolve the retention conflict. If permission is unavailable,
reopen source selection; do not delete history or silently redesign the product
to fit a temporary cache. [Best Buy API terms](https://developer.bestbuy.com/legal)

## Candidates and evidence

### Best Buy US — preferred technical target, access/retention gate

- **Access:** official Products API, JSON/XML, individual SKU lookup. Registration
  and activation of an API key are documented; key issuance and a live response
  were not tested. Public documentation is not proof that our account has access.
- **Fields:** `salePrice` means current selling price, `regularPrice` regular
  selling price, and `onSale` identifies a reduction. `onlineAvailability` describes
  online purchase availability. `priceRestriction` identifies MAP/checkout
  restrictions. `offers` provides descriptive promotion data and dates. No explicit
  MSRP field was established in the reviewed pricing documentation.
- **Structure/testing:** documented JSON is preferable to website JSON-LD or
  embedded page state. Examples support synthetic fixtures; no stable public-page
  parsing contract or JSON-LD coverage was verified. Documentation contains older
  examples, so live field behavior needs a permitted smoke check.

Evidence: [official API documentation](https://bestbuyapis.github.io/api-documentation/).

- **Limits/policy:** published Products limit is 5 calls/second and 50,000/day per
  key; quota excess can return 403. Terms require attribution, constrain usage and
  supplied links, restrict third-party pricing-analysis uses, and allow termination.
  The 72-hour storage limit is the principal blocker for this project.
  [API terms and operational policy](https://developer.bestbuy.com/legal)
- **Public site:** the wildcard robots group excludes internal pricing,
  availability, cart, and other service paths. Do not substitute those endpoints
  for licensed API access. Robots rules are not a license to retain data.
  [Best Buy robots.txt](https://www.bestbuy.com/robots.txt)
- **Assessment:** lowest parser-maintenance burden among candidates; permission,
  access continuity, and data freshness remain material risks. A documented API
  is not an availability guarantee.

### Dell US official store — defer

- **Official API/access:** Dell's Catalog API returns organization-specific
  pricing/configuration in JSON/XML. It requires a Premier page, procurement
  integration, Dell-team activation, and OAuth. Annual credential maintenance is
  documented. This is not a verified self-service public consumer-price API.
  No numeric quota was established in the reviewed overview; confirm it with Dell
  if this route is revisited.
  [Dell Catalog API](https://developer.dell.com/apis/42342cfa-92ad-48f2-818c-b9922ac24e8e/versions/2.0.0)
- **Public-site constraints:** Dell's AUP prohibits high-volume automated access;
  its robots file also excludes internal API/cart/configuration paths. Neither
  establishes permission for this tracker or indefinite price-history retention.
  [Dell AUP](https://www.dell.com/en-us/lp/legal/acceptable-use-policy),
  [robots.txt](https://www.dell.com/robots.txt)
- **Data:** the public store displays Dell Price, offer/configuration IDs, savings,
  and Estimated Value. Dell describes Estimated Value as a market comparison
  estimate, so it must not be relabeled MSRP or a prior selling price. Public-page
  stock semantics, an explicit MSRP field, and a stable JSON-LD/embedded-data
  contract were not verified. Promotions are visible, but no permitted stable
  consumer promotion feed was established.
  [Dell US laptop store and pricing explanation](https://www.dell.com/en-us/shop/dell-laptops/scr/laptop)
- **Assessment:** negotiated access and customer/configuration-specific prices
  make development less practical. Public-page parsing would add unverified
  permission and maintenance risks. Revisit only with an authorized feed and
  precisely defined consumer or business context.

### Amazon US — reject for the first provider

- **Access:** current official documentation describes Creators API, Associates
  enrollment, registration/credentials, and at least 10 qualifying sales in the
  preceding 30 days for PA API access through Creators API. Do not assume legacy
  PA-API onboarding is still the right path.
  [Creators API prerequisites](https://affiliate-program.amazon.com/creatorsapi/docs/)
- **Limits:** documentation lists an initial maximum of 1 request/second and
  8,640/day for 30 days, then revenue-dependent allocations; continued access
  depends on qualifying referrals. Account eligibility was not tested.
  [API rates](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/concepts/api-rates)
- **Data:** OffersV2 documents currency-bearing prices, availability, merchant and
  condition context, saving-basis labels, and deal details. A saving basis is not
  automatically MSRP. Customer-visible prices can differ from returned prices.
  Official JSON is preferable to unverified website JSON-LD or embedded data.
  [OffersV2](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/api-reference/resources/offersV2)
- **Policy:** Associates policy clause (y) prohibits price tracking/alerts unless
  Amazon agrees otherwise. The license also restricts extraction tools and how
  content is used to direct traffic. Website scraping is not a fallback.
  [Associates policies](https://affiliate-program.amazon.com/help/operating/policies)
- **Assessment:** explicit product-purpose restriction, sales-gated access, and
  seller/condition context make it unsuitable for this bootstrap. Rich data does
  not outweigh those constraints; no implementation is selected.

## Proposed narrow Best Buy scope after the gates are satisfied

These are project decisions, not claims that permission has been granted:

- One explicitly registered US Listing and exact Best Buy SKU per request, using
  the official Products API only. Require the SKU in `RetailerProductID`; verify
  response identity. Support only confirmed Best Buy-sold, new, ordinary products.
  Exclude ambiguous seller/condition contexts rather than guessing.
- US online purchasing, USD only. Confirm currency context during authorized
  validation; a dollar symbol alone is insufficient. Convert decimal amounts
  exactly to integer minor units; missing remains missing and explicit zero stays zero.
- Map current `salePrice` to OfferPrice unless `onSale` explicitly establishes
  sale semantics; then use SalePrice. Map `regularPrice` to RetailerListPrice,
  never MSRP. Keep MSRP and MSRPSource absent initially.
- Interpret stock only within online purchase context: explicit availability
  supports in-stock; explicit unavailability supports out-of-stock only for the
  supported ordinary-product context. Missing/ambiguous availability stays unknown.
  Do not infer local stock, quantities, or delivery promises. Reject unsupported
  preorder/backorder/special-order context rather than flattening its meaning.
- Restricted-price responses are unsupported; do not infer hidden checkout prices.
  Preserve a non-secret source reference and actual observation time. Never store
  an API key or credential-bearing request URL in provenance or fixtures.
- No promotions, membership/coupon/cashback arithmetic, financing, trade-ins,
  bundles, open-box, marketplace sellers, store pickup, or automatic product matching.
  Promotion fields may exist upstream but are outside Task 14.
- Preserve the Provider context/result/error contract. Authentication, quota,
  missing-SKU, malformed-response, and access failures must remain collection errors,
  not empty successful observations. Do not add bypasses, hidden endpoints, CAPTCHA
  workarounds, key rotation to evade quotas, or a website-scraping fallback.

## Task 14/15 handoff and unresolved evidence

Task 14 can prepare a narrow offline adapter/parser with synthetic fixtures based
on documented shapes; it must not be described as production-ready. Fixtures should
cover distinct price meanings, missing/zero amounts, stock-only results, identity
mismatch, restricted contexts, malformed data, and failures. Do not commit live
response snapshots under an assumed permanent-storage entitlement.

Before live validation, record the applicable permission, verified account/key
availability, permitted retention/display terms, supported currency/seller context,
and actual field behavior. Consult the [official contact route](https://developer.bestbuy.com/contact-us)
for the permission question; no message or registration was submitted in Task 13.
If these gates fail, stop the live-provider path and revisit this decision.

No Go interface incompatibility was found for the proposed restricted context.
The concrete incompatibility is between default source retention terms and the
project's permanent observations, not a reason to change the Provider interface.
Existing roadmap phases and architecture remain unchanged.

## Verification boundaries

Official documentation/policy pages and robots files linked above were retrieved
or surfaced by current official-source search on the research date. Links can
redirect; recheck terms before implementation and live use. API credentials,
authenticated calls, retailer HTML source schemas, and production permissions
were not verified. An unavailable field or permission is explicitly unknown,
not evidence that the field cannot exist or that access is allowed.
