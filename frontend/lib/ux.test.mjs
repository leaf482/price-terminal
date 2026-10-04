import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {displayPrice,formatPrice,basis} from './prices.ts';

test('display boundary preserves missing/zero and reports malformed prices without inventing facts',()=>{
 assert.equal(basis(null),null);assert.equal(basis({stock:'unknown'}),null);
 assert.equal(displayPrice(0,'USD'),'USD 0.00');assert.equal(displayPrice(0,'JPY'),'JPY 0');
 for(const [amount,currency] of [[NaN,'USD'],[undefined,'USD'],[null,'USD'],[-1,'JPY'],[1,'EUR'],[1,undefined],[Number.MAX_SAFE_INTEGER+1,'USD']])assert.equal(displayPrice(amount,currency),'Invalid price data');
 assert.throws(()=>formatPrice(1,'EUR'));
});
test('global navigation provides a focusable skip target and every primary destination',()=>{
 const exports={},require=createRequire(import.meta.url);const code=ts.transpileModule(readFileSync(new URL('../app/layout.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
 runInNewContext(code,{exports,require(name){if(name==='next/link')return{default:({href,children,...props})=>React.createElement('a',{href,...props},children)};if(name.endsWith('.css'))return{};if(name==='./global-search')return{default:()=>null};return require(name)}});
 const html=renderToStaticMarkup(exports.default({children:React.createElement('main',null,'Page')}));
 for(const path of ['/','/products','/catalog','/alerts','/collection','/price-changes','#page-content'])assert.ok(html.includes(`href="${path}"`));assert.match(html,/id="page-content" tabindex="-1"/);assert.match(html,/aria-label="Main navigation"/);
});
test('dense tables expose keyboard scrolling, header scope and narrow-screen styles',()=>{
 for(const [path,label] of [['../app/collection/page.tsx','Collection health table'],['../app/price-changes/page.tsx','Price changes table'],['../app/products/[id]/quality-view.tsx','Observation audit table']]){
  const source=readFileSync(new URL(path,import.meta.url),'utf8');assert.ok(source.includes(`className="table-scroll" role="region" aria-label="${label}" tabIndex={0}`));assert.match(source,/<th scope="col"/);
 }
 const css=readFileSync(new URL('../app/globals.css',import.meta.url),'utf8');assert.match(css,/\.primary-nav[^}]*flex-wrap: wrap/);assert.match(css,/\.table-scroll[^}]*overflow-x: auto/);assert.match(css,/:focus-visible/);assert.match(css,/@media \(max-width: 40rem\)/);
});
test('actions, loading and result announcements retain explicit semantic labels',()=>{
 const read=p=>readFileSync(new URL(p,import.meta.url),'utf8');
 assert.match(read('../app/products/[id]/alert-view.tsx'),/aria-label=.*alert \$\{a.id\}/);
 assert.match(read('../app/products/[id]/record-price.tsx'),/<label>Observed time/);
 assert.match(read('../app/collection/page.tsx'),/role="status">\{results.length\} collection results/);
 assert.match(read('../app/global-search.tsx'),/role="status">\{data.products.length/);
 assert.match(read('../app/listings/[id]/attempt-history.tsx'),/role="status">Loading collection/);
 const listing=read('../app/listings/[id]/page.tsx');assert.match(listing,/Observed current price/);assert.match(listing,/Tracking:/);assert.match(listing,/Freshness:/);
 const effective=read('../app/products/[id]/promotion-view.tsx');assert.match(effective,/Derived EffectivePrice/);assert.match(effective,/Possible cashback/);
});

test('visual treatments retain conditional-price language and complete audit facts',()=>{
 const read=p=>readFileSync(new URL(p,import.meta.url),'utf8');
 const effective=read('../app/products/[id]/promotion-view.tsx');
 assert.match(effective,/panel derived-price/);
 assert.match(effective,/Conditional savings and cashback are not guaranteed/);
 const audit=read('../app/products/[id]/quality-view.tsx');
 assert.ok(audit.includes('className="timestamp">{o.observed_at}'));
 assert.ok(audit.includes('className="numeric">{price(a,o.offer_price)}'));
 assert.ok(audit.includes("{a.valid?'Valid':'Invalidated'}"));
 const css=read('../app/globals.css');
 assert.match(css,/\.numeric[^}]*text-align: right/);
 assert.match(css,/\.catalog-search form[^}]*flex-wrap: wrap/);
 assert.match(css,/overflow-wrap: anywhere/);
 assert.match(css,/input, select, textarea[^}]*max-width: 100%/);
});

test('consumer hierarchy places history before maintenance and retains advanced tools',()=>{
 const read=p=>readFileSync(new URL(p,import.meta.url),'utf8');
 const product=read('../app/products/[id]/page.tsx');
 assert.ok(product.indexOf('className="price-hero"') < product.indexOf('<HistoryView'));
 assert.ok(product.indexOf('<HistoryView') < product.indexOf('<ListingComparison prices='));
 assert.ok(product.indexOf('<ListingComparison prices=') < product.indexOf('<ArchiveControls'));
 const listing=read('../app/listings/[id]/page.tsx');
 for(const tool of ['RecordPrice','CSVImport','CSVExport','QualityView','AttemptHistory']) assert.ok(listing.indexOf('<HistoryView') < listing.indexOf('<'+tool+' '));
 assert.match(read('../app/products/[id]/listing-comparison.tsx'), /<summary>Listing tools &amp; audit<\/summary>/);
 assert.match(read('../app/layout.tsx'), /href="#catalog-search"/);
});
