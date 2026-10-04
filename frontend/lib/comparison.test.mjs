import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as comparison from './comparison.ts';
import * as prices from './prices.ts';
import {parseRetailers} from './catalog.ts';

const make=(id,currency,amount,time='2026-10-01T00:00:00Z')=>({listing:{id,product_id:'p',retailer_id:id,url:'https://example.com/'+id,tracking_enabled:true},observation:amount===null?null:{observed_at:time,source:'fixture',stock:'in_stock',currency,offer_price:amount},freshness:'fresh',collection:{state:'inactive'}});
const zero=make('a','USD',0), usd=make('b','USD',100), jpy=make('c','JPY',200), missing=make('d',null,null);
const names={a:'Zulu',b:'Alpha',c:'Beta',d:'Delta'};
test('observed price sorting groups currencies, preserves zero/missing and never uses effective price',()=>{
 const rows=[missing,usd,jpy,zero],copy=[...rows];usd.effective_price=-1;
 const ids=mode=>comparison.sortListings(rows,mode,names).map(r=>r.listing.id);
 assert.deepEqual(ids('price_asc'),['c','a','b','d']);assert.deepEqual(ids('price_desc'),['c','b','a','d']);assert.deepEqual(ids('retailer'),['b','c','d','a']);assert.deepEqual(rows,copy);
 assert.deepEqual(comparison.sortListings([usd,zero],'price_asc',names),[zero,usd]);
 const sale={...usd,observation:{...usd.observation,offer_price:undefined,sale_price:50}};
 assert.deepEqual(comparison.sortListings([sale,zero],'price_desc',names),[sale,zero]);
});
test('newest sort handles nanoseconds, offsets, ties, stock-only and missing observations',()=>{
 const early=make('e','USD',1,'2026-10-01T00:00:00.123456788Z'), late=make('f','USD',1,'2026-10-01T01:00:00.123456789+01:00');
 const stock={...make('s','USD',1,'2026-10-02T00:00:00Z'),observation:{observed_at:'2026-10-02T00:00:00Z',source:'fixture',stock:'unknown'}};
 assert.deepEqual(comparison.sortListings([missing,early,late,stock],'newest',names).map(r=>r.listing.id),['s','f','e','d']);
 assert.deepEqual(comparison.sortListings([usd,zero],'newest',names).map(r=>r.listing.id),['a','b']);
});
function load(retailerNames=names){
 const states=[];let index=0;
 const exports={},require=createRequire(import.meta.url),hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]}};
 const code=ts.transpileModule(readFileSync(new URL('../app/products/[id]/listing-comparison.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,require(name){if(name==='react')return hooks;if(name.endsWith('/comparison'))return comparison;if(name.endsWith('/prices'))return prices;if(name.startsWith('./'))return{default:({listingID,id})=>React.createElement('span',{'data-control':name},listingID||id)};return require(name)}});
 return data=>{index=0;return exports.default({prices:data,retailers:retailerNames,revision:'revision'})};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}

test('prototype-named Retailers outside the initial batch are fetched, rendered and sorted as string data',async()=>{
 for(const id of ['constructor','__proto__','toString','normal-outside']){
  const rows=[make(id,'USD',100),zero],data={product_id:'p',listings:rows,best_price:null,comparison_status:'not_fresh'};
  const batch=[{id:'a',name:'Zulu shop'},...Array.from({length:99},(_,i)=>({id:`other-${i}`,name:`Other ${i}`}))];
  const calls=[],exports={},require=createRequire(import.meta.url);
  const component=()=>null;
  const code=ts.transpileModule(readFileSync(new URL('../app/products/[id]/page.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
  runInNewContext(code,{exports,require(name){
   if(name==='../../../lib/catalog')return{parseRetailers};
   if(name==='../../../lib/prices')return prices;
   if(name==='../../../lib/api')return{parseProduct:x=>x,parsePrices:x=>x,api:async(path,parse)=>{calls.push(path);const value=path==='/products/p'?{id:'p',name:'Product',brand:'',model:''}:path==='/products/p/prices'?data:path==='/retailers?limit=100'?batch:path===`/retailers/${id}`?{id,name:'Alpha real shop'}:assert.fail(`Unexpected request ${path}`);return parse(value)}};
   if(name==='next/link'||name.startsWith('./')||name.startsWith('../../catalog/'))return{default:component};return require(name);
  }});
  const tree=await exports.default({params:Promise.resolve({id:'p'})});
  const props=nodes(tree).find(n=>n.props?.retailers)?.props;
  assert.ok(props);assert.equal(calls.filter(p=>p===`/retailers/${id}`).length,1);assert.ok(!calls.includes('/retailers/a'));
  assert.ok(Object.hasOwn(props.retailers,id));assert.equal(typeof props.retailers[id],'string');assert.equal(props.retailers[id],'Alpha real shop');
  const render=load(props.retailers);let comparisonTree=render(data);
  assert.match(renderToStaticMarkup(comparisonTree),/<h3>Alpha real shop<\/h3>/);
  assert.match(renderToStaticMarkup(comparisonTree),/<h3>Zulu shop<\/h3>/);
  nodes(comparisonTree).find(n=>n.type==='select').props.onChange({target:{value:'retailer'}});
  comparisonTree=render(data);assert.deepEqual(nodes(comparisonTree).filter(n=>n.type==='article').map(n=>n.props.id),[id,'a']);
 }
 assert.throws(()=>parseRetailers([{id:'constructor',name:()=>{}}]),/Invalid catalog record/);
});
test('comparison cards retain status, best-price contract, promotion separation and all controls while sorting',()=>{
 const rows=[{...usd,freshness:'stale',collection:{state:'failed'},observation:{...usd.observation,stock:'out_of_stock'}},{...zero,listing:{...zero.listing,tracking_enabled:false}},missing,jpy];
 const data={product_id:'p',listings:rows,best_price:null,comparison_status:'incompatible_currencies'},render=load();
 let tree=render(data),html=renderToStaticMarkup(tree);
 assert.match(html,/href="\/listings\/a"/);
 for(const text of ['Alpha','Observed current price','USD 0.00','Price unavailable','out of stock','Tracking: disabled','Collection: failed','Observation freshness: stale','incompatible currencies','Conditional savings are not guaranteed','EffectivePrice are excluded'])assert.ok(html.includes(text),text);
 for(const control of ['promotion-view','alert-view','record-price','quality-view','csv-import','csv-export','tracking-controls','collection-controls'])assert.match(html,new RegExp('data-control="./'+control+'"'));
 for(const mode of ['price_asc','price_desc','newest','retailer']){nodes(tree).find(n=>n.type==='select').props.onChange({target:{value:mode}});tree=render(data);assert.deepEqual(nodes(tree).filter(n=>n.type==='article').map(n=>n.props.id),comparison.sortListings(rows,mode,names).map(r=>r.listing.id))}
 const valid={...data,listings:[zero,usd],best_price:{minor_units:0,currency:'USD',basis:'offer_price',listing_id:'a'},comparison_status:'comparable'};
 assert.match(renderToStaticMarkup(render(valid)),/Best comparable observed price:.*USD 0.00/);
 for(const reason of ['not_fresh','not_in_stock','missing_price','incompatible_currencies'])assert.ok(renderToStaticMarkup(render({...data,comparison_status:reason})).includes('Unavailable — '+reason.replaceAll('_',' ')));
});
