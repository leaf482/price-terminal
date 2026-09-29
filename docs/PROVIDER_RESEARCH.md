# Best Buy access feasibility

Checked: 2026-09-28. Outcome: **BLOCKED under the published API terms**.

This Task 14 feasibility review supersedes the conditional implementation handoff
in [the Task 13 decision](FIRST_PROVIDER_DECISION.md). The requested feasibility
gate takes precedence over the adapter work listed in [TASKS.md](TASKS.md).
No adapter is approved by this review; roadmap numbering is unchanged.

## Access: documented route versus verified issuance

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

## Data and exact endpoint

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

## Historical retention: the blocking condition

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

## Next action and reopening criteria

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

## Verification boundary

Official sources above were opened on the check date; the login/account UI was
inspected without submission. Published policy and documented fields are
distinguished from unverified issuance and live data. No credentials, code,
dependencies, or provider interface changes are part of this review.
