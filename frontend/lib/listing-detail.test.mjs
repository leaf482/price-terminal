import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {loadListingDetail} from './listing-detail.ts';
import * as prices from './prices.ts';

const current={listing:{id:'constructor',product_id:'__proto__',retailer_id:'constructor',url:'https://example.com/item',retailer_product_id:'SKU',tracking_enabled:false},observation:{observed_at:'2026-01-01T00:00:00.123456789Z',source:'manual: fixture',stock:'in_stock',currency:'USD',offer_price:0},freshness:'stale',collection:{state:'failed',error:'collection_failed',last_attempted_at:'2026-10-01T00:00:00Z',last_successful_at:'2026-09-01T00:00:00Z'}};
const product={id:'__proto__',name:'Camera',brand:'Maker',model:'Model'},retailer={id:'constructor',name:'Shop'};
test('Listing load uses existing endpoints, handles missing related records and never masks outages',async()=>{
 const original=globalThis.fetch;let calls=[];
 try{
  globalThis.fetch=async url=>{const path=new URL(url).pathname;calls.push(path);const value=path==='/listings/constructor/price'?current:path==='/products/__proto__'?product:path==='/retailers/constructor'?retailer:assert.fail(path);return Response.json({data:value})};
  assert.deepEqual(await loadListingDetail('constructor'),{current,product,retailer});assert.equal(calls.length,3);
  calls=[];globalThis.fetch=async url=>{calls.push(url);return new Response('',{status:404})};assert.equal(await loadListingDetail('missing'),null);assert.equal(calls.length,1);
  globalThis.fetch=async url=>url.includes('/listings/')?Response.json({data:current}):new Response('',{status:404});assert.deepEqual(await loadListingDetail('constructor'),{current,product:null,retailer:null});
  globalThis.fetch=async url=>url.includes('/listings/')?Response.json({data:current}):new Response('',{status:500});await assert.rejects(loadListingDetail('constructor'),/500/);
 }finally{globalThis.fetch=original}
});
function load(file,data){
 const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL(file,import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,require(name){
  if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};
  if(name==='next/navigation')return{notFound(){throw new Error('not-found')}};
  if(name.endsWith('/listing-detail'))return{loadListingDetail:async id=>{assert.equal(id,'constructor');return data}};
  if(name.endsWith('/prices'))return prices;
  if(name === './attempt-history' || name.startsWith('../../products/'))return{default:props=>React.createElement('span',{'data-control':name},JSON.stringify(props))};
  return require(name);
 }});return exports.default;
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('Listing composition preserves navigation, price/stock states and existing tools',async()=>{
 for(const observation of [current.observation,null,{observed_at:'2026-01-01T00:00:00Z',source:'fixture',stock:'unknown'}]){
  const c={...current,observation};const page=load('../app/listings/[id]/page.tsx',{current:c,product,retailer});const tree=await page({params:Promise.resolve({id:'constructor'})});const html=renderToStaticMarkup(tree);
  for(const text of ['Camera','Maker','Model','Shop','SKU','Tracking: disabled','Freshness: stale'])assert.ok(html.includes(text),text);
  assert.match(html,/href="\/products\/__proto__"/);assert.match(html,/href="\/retailers\/constructor"/);
  assert.match(html,observation?.offer_price===0?/USD 0.00/:/Price unavailable/);
  if(observation)assert.ok(html.includes(observation.observed_at));
  for(const control of ['collection-controls','tracking-controls','record-price','csv-import','csv-export','history-view','promotion-view','alert-view','quality-view'])assert.ok(html.includes(`products/[id]/${control}`));
  const collection=nodes(tree).find(n=>n.props?.status);assert.equal(collection.props.trackingEnabled,false);assert.deepEqual(collection.props.status,current.collection);
  const history=nodes(tree).find(n=>n.props?.listings);assert.equal(history.props.listings[0].id,'constructor');
  assert.ok(nodes(tree).some(n=>n.props?.table===true&&n.props.listingID==='constructor'));
 }
 const missing=load('../app/listings/[id]/page.tsx',null);await assert.rejects(missing({params:Promise.resolve({id:'constructor'})}),/not-found/);
 const fallback=await load('../app/listings/[id]/page.tsx',{current,product:null,retailer:null})({params:Promise.resolve({id:'constructor'})});assert.match(renderToStaticMarkup(fallback),/Product metadata is unavailable/);assert.match(renderToStaticMarkup(fallback),/Retailer metadata is unavailable/);
 assert.match(renderToStaticMarkup(load('../app/listings/[id]/loading.tsx')()),/Loading Listing detail/);
 assert.match(renderToStaticMarkup(load('../app/listings/[id]/error.tsx')({reset(){}})),/role="alert"/);
 assert.match(renderToStaticMarkup(load('../app/listings/[id]/not-found.tsx')()),/Listing not found/);
});
