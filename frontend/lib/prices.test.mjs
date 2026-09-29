import {test} from 'node:test';
import assert from 'node:assert/strict';
import {parseObservation,parseHistory,parseProducts,parsePrices,api} from './api.ts';
import {formatPrice,historyPoints,historySegments,basis,safeSource} from './prices.ts';
const row={observed_at:'2026-09-29T00:00:00Z',source:'fixture',stock:'unknown',currency:'USD',offer_price:0};
test('exact formatting and zero',()=>{assert.equal(formatPrice(1999,'USD'),'USD 19.99');assert.equal(formatPrice(0,'USD'),'USD 0.00');assert.equal(formatPrice(123,'JPY'),'JPY 123');assert.equal(formatPrice(Number.MAX_SAFE_INTEGER,'USD'),'USD 90071992547409.91');assert.throws(()=>formatPrice(Number.MAX_SAFE_INTEGER+1,'USD'));});
test('API validation rejects unsafe prices and preserves absent money',()=>{assert.equal(parseObservation(row).offer_price,0);assert.throws(()=>parseObservation({...row,offer_price:1.5}));assert.throws(()=>parseObservation({...row,offer_price:9007199254740992}));assert.throws(()=>parseObservation({...row,currency:'EUR'}));assert.deepEqual(parseProducts([]),[]);assert.throws(()=>parseProducts([{}]));});
test('history preserves missing points, zero and currencies without filling',()=>{const stock={observed_at:'2026-09-29T01:00:00Z',source:'fixture',stock:'unknown'};const observations=[row,stock,{...row,currency:'JPY',observed_at:'2026-09-29T02:00:00Z'}];const history=parseHistory({listing_id:'l',observations,truncated:false,historical_low:null,period_change_percent:null});const points=historyPoints(history.observations);assert.equal(points.length,3);assert.equal(points[0].price.amount,0);assert.equal(points[1].price,null);assert.equal(points[2].price.currency,'JPY');assert.equal(basis({...row,sale_price:1}).amount,0);assert.equal(basis({...stock,sale_price:2,currency:'USD'}).amount,2);assert.equal(basis(stock),null);});
test('empty history and current data',()=>{assert.deepEqual(parseHistory({listing_id:'l',observations:[],truncated:false,historical_low:null,period_change_percent:null}).observations,[]);assert.equal(parsePrices({product_id:'p',listings:[],best_price:null,comparison_status:'no_listings'}).best_price,null);assert.equal(safeSource('javascript:alert(1)'),null);});
test('API success and failure',async(t)=>{t.mock.method(globalThis,'fetch',async()=>new Response(JSON.stringify({data:[]})));assert.deepEqual(await api('/products',parseProducts),[]);globalThis.fetch=async()=>new Response('',{status:500});await assert.rejects(api('/products',parseProducts),/500/);});
test('line segments break at missing prices, long gaps and currency/basis changes',()=>{
 const next={...row,observed_at:'2026-09-29T00:01:00Z'};
 assert.equal(historySegments([row,next])[0].length,2);
 const stock={observed_at:'2026-09-29T00:00:30Z',source:'fixture',stock:'unknown'};
 assert.deepEqual(historySegments([row,stock,next]).map(x=>x.length),[1,1]);
 assert.equal(historySegments([row,{...next,observed_at:'2026-09-29T02:00:00Z'}]).length,2);
 assert.equal(historySegments([row,{...next,currency:'JPY'}]).length,2);
 assert.equal(historySegments([row,{...next,offer_price:undefined,sale_price:0}]).length,2);
});
