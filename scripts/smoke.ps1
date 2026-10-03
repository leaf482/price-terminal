# Opt-in: creates uniquely named catalog/observation records; does not delete data.
# Requires PowerShell 7+, and the running migrated local backend.
param([string]$BaseUrl = 'http://127.0.0.1:8080')
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$base = [Uri]$BaseUrl
if (-not $base.IsAbsoluteUri -or $base.Scheme -notin @('http','https') -or $base.UserInfo -or $base.Query -or $base.Fragment -or $base.AbsolutePath -ne '/') { throw 'BaseUrl must be an HTTP(S) origin without credentials or path.' }
$BaseUrl = $BaseUrl.TrimEnd('/')
$prefix = 'smoke-' + [Guid]::NewGuid().ToString('N')
Write-Host "Smoke prefix: $prefix (records remain, even after a partial failure)"
function Request([string]$method, [string]$path, [int]$expected, $body = $null) {
    $options = @{ Uri = "$BaseUrl$path"; Method = $method; TimeoutSec = 15; SkipHttpErrorCheck = $true; MaximumRedirection = 0 }
    if ($null -ne $body) { $options.ContentType = 'application/json'; $options.Body = $body | ConvertTo-Json -Depth 6 -Compress }
    try { $response = Invoke-WebRequest @options } catch { throw "$method $path transport failure; check the running backend. Prefix: $prefix" }
    if ([int]$response.StatusCode -ne $expected) { throw "$method $path expected HTTP $expected, got $($response.StatusCode). Prefix: $prefix" }
    return $response.Content
}
function Check([bool]$ok,[string]$message) { if (-not $ok) { throw "Smoke assertion failed: $message. Prefix: $prefix" } }
Check ((Request GET /healthz 200).Trim() -eq 'ok') 'health response'
Check ((Request GET /readyz 200).Trim() -eq 'ready') 'readiness response'
$p = (Request POST /products 201 @{id="$prefix-product"; name='Smoke product'; brand='Smoke'; model='Manual demo'} | ConvertFrom-Json).data
Check ($p.id -ceq "$prefix-product") 'Product identity'
$r = (Request POST /retailers 201 @{id="$prefix-retailer"; name='Smoke retailer'} | ConvertFrom-Json).data
Check ($r.id -ceq "$prefix-retailer") 'Retailer identity'
$l = (Request POST /listings 201 @{id="$prefix-listing"; product_id=$p.id; retailer_id=$r.id; url="https://example.com/$prefix"} | ConvertFrom-Json).data
Check ($l.id -ceq "$prefix-listing") 'Listing identity'
$time = [DateTimeOffset]::UtcNow.AddSeconds(-1).ToString('o')
$o = (Request POST "/listings/$($l.id)/observations" 201 @{id="$prefix-observation"; observed_at=$time; source='opt-in local smoke test'; stock='in_stock'; offer_price=@{minor_units=1999;currency='USD'}} | ConvertFrom-Json).data
Check ($o.id -ceq "$prefix-observation") 'observation identity'
$current = (Request GET "/listings/$($l.id)/price" 200 | ConvertFrom-Json).data
Check ($current.listing.id -ceq $l.id -and $current.observation.offer_price -eq 1999 -and $current.observation.currency -ceq 'USD' -and $current.observation.stock -ceq 'in_stock') 'current observation values'
Check ([DateTimeOffset]$current.observation.observed_at -eq [DateTimeOffset]$time) 'observation timestamp'
Check ($current.observation.source -ceq 'manual: opt-in local smoke test') 'manual provenance'
$history = (Request GET "/listings/$($l.id)/history?range=ALL" 200 | ConvertFrom-Json).data
Check ($history.listing_id -ceq $l.id -and @($history.observations).Count -eq 1 -and -not $history.truncated) 'history identity/count'
Check ($history.observations[0].offer_price -eq 1999 -and $history.observations[0].currency -ceq 'USD' -and [DateTimeOffset]$history.observations[0].observed_at -eq [DateTimeOffset]$time) 'history values'
Write-Host "PASS: health, readiness, catalog creation, manual observation, current price and history."
Write-Host "Open http://localhost:3000/listings/$($l.id) (adjust frontend origin if needed)."
