# Deterministic script-contract checks; no network or database access.
$ErrorActionPreference = 'Stop'
$state = @{count=0;bad=$false;badStatus=$false;product=$null;retailer=$null;listing=$null;observation=$null}
function Invoke-WebRequest {
    param($Uri,$Method,$TimeoutSec,$SkipHttpErrorCheck,$MaximumRedirection,$ContentType,$Body)
    $state.count++
    $path = ([Uri]$Uri).AbsolutePath
    if ($state.badStatus) { return @{StatusCode=503;Content='not ready'} }
    if ($path -eq '/healthz') { return @{StatusCode=200;Content='ok'} }
    if ($path -eq '/readyz') { return @{StatusCode=200;Content='ready'} }
    if ($Method -eq 'POST') {
        $v = $Body | ConvertFrom-Json
        if ($path -eq '/products') { $state.product = $v }
        if ($path -eq '/retailers') { $state.retailer = $v }
        if ($path -eq '/listings') {
            if ($v.product_id -ne $state.product.id -or $v.retailer_id -ne $state.retailer.id) { throw 'Wrong relationships' }
            $state.listing = $v
        }
        if ($path.EndsWith('/observations')) {
            $state.observation = @{observed_at=$v.observed_at;source='manual: '+$v.source;stock=$v.stock;currency=$v.offer_price.currency;offer_price=$v.offer_price.minor_units}
        }
        return @{StatusCode=201;Content=(@{data=$v}|ConvertTo-Json -Depth 6)}
    }
    if ($path.EndsWith('/price')) {
        if ($state.bad) { $state.observation.offer_price = 123 }
        return @{StatusCode=200;Content=(@{data=@{listing=$state.listing;observation=$state.observation}}|ConvertTo-Json -Depth 6)}
    }
    if ($path.EndsWith('/history')) { return @{StatusCode=200;Content=(@{data=@{listing_id=$state.listing.id;observations=@($state.observation);truncated=$false}}|ConvertTo-Json -Depth 6)} }
    throw 'Unexpected smoke request'
}
& "$PSScriptRoot/smoke.ps1"
if ($state.count -ne 8) { throw "Unexpected request count: $state.count" }
$first = $state.product.id
$state.bad = $true
$rejected = $false
try { & "$PSScriptRoot/smoke.ps1" } catch { $rejected = $_.Exception.Message -like '*current observation values*' }
if (-not $rejected -or $first -eq $state.product.id) { throw 'Smoke must reject wrong data and use unique IDs' }
$state.badStatus = $true
$rejected = $false
try { & "$PSScriptRoot/smoke.ps1" } catch { $rejected = $_.Exception.Message -like '*expected HTTP 200, got 503*' }
if (-not $rejected) { throw 'Smoke must reject unexpected HTTP status' }
Write-Host 'PASS: smoke script contract tests (mock transport only).'
