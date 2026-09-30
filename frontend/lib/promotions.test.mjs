import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import {renderToStaticMarkup} from 'react-dom/server';
import * as promotions from './promotions.ts';
import * as prices from './prices.ts';

const p={id:'p',listing_id:'l',source:'fixture',observed_at:'2026-09-29T00:00:00Z',kind:'fixed',amount:{minor_units:100,currency:'USD'},requirement:'unknown',stacking:'unknown',terms:'Coupon CODE, eligibility unknown'};
const amount=n=>({minor_units:n,currency:'USD'});
const result={status:'available',reason:'supported_scenario',base:amount(1000),immediate_discount:amount(100),immediate_payable:amount(900),potential_cashback:amount(50),potential_net:amount(850),observation:{observed_at:p.observed_at,source:'fixture',stock:'in_stock',offer_price:1000,currency:'USD'},calculated_at:p.observed_at,assumptions:['Eligibility is assumed; cashback is not guaranteed.'],exclusions:['tax','shipping']};
function summary(data){
 const source=readFileSync(new URL('../app/products/[id]/promotion-view.tsx',import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 const require=createRequire(import.meta.url),exports={};
 runInNewContext(compiled,{exports,require(name){if(name==='../../../lib/api')return {};if(name==='../../../lib/prices')return prices;if(name==='../../../lib/promotions')return promotions;return require(name)}});
 return renderToStaticMarkup(exports.EffectiveSummary({data}));
}
test('promotion evidence preserves unknown conditions and percentage precision',()=>{
 assert.equal(promotions.parsePromotions({promotions:[p],truncated:false}).promotions[0].requirement,'unknown');
 assert.equal(promotions.parsePromotions({promotions:[{...p,kind:'percentage',amount:undefined,basis_points:1250}],truncated:false}).promotions[0].basis_points,1250);
 assert.throws(()=>promotions.parsePromotions({promotions:[{...p,amount:amount(Number.MAX_SAFE_INTEGER+1)}],truncated:false}));
});
test('derived summary distinguishes observed, payable and possible cashback/net',()=>{
 const data=promotions.parseEffective(result),html=summary(data);
 for(const text of ['Base observed price','Derived immediate payable','Possible cashback','Potential net after cashback','USD 10.00','USD 9.00','USD 0.50','USD 8.50','not guaranteed','tax, shipping']) assert.ok(html.includes(text),text);
});
for(const status of ['conditional','unavailable'])test(`${status} summaries withhold totals`,()=>{
 const data={...result,status,reason:'unknown_eligibility',immediate_discount:null,immediate_payable:null,potential_cashback:null,potential_net:null};
 const html=summary(promotions.parseEffective(data));assert.match(html,/Derived totals are withheld/);assert.doesNotMatch(html,/Derived immediate payable:/);
 assert.throws(()=>promotions.parseEffective({...data,immediate_payable:amount(0)}));
});
test('explicit zero derived result remains valid',()=>{assert.equal(promotions.parseEffective({...result,immediate_payable:amount(0),potential_cashback:amount(0),potential_net:amount(0)}).potential_net.minor_units,0)});
