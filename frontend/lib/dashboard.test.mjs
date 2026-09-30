import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import * as dashboard from './dashboard.ts';
import * as prices from './prices.ts';

const zero={product:{id:'p / zero',name:'Camera',brand:'Acme',model:'Alpha'},listing_count:2,has_current_price:true,best_price:{minor_units:0,currency:'USD',basis:'offer_price',listing_id:'l'},comparison_status:'comparable',latest_observation_at:'2026-09-30T10:00:00Z',latest_attempt_at:'2026-09-30T12:00:00Z',freshness:{fresh:2},collection:{failed:1,inactive:1},has_collection_error:true,recent_alert:true};
const missing={...zero,product:{id:'missing',name:'Keyboard',brand:'Keys',model:'Beta'},has_current_price:false,best_price:null,comparison_status:'missing_observation',latest_observation_at:null,latest_attempt_at:null,freshness:{missing:2},collection:{inactive:2},has_collection_error:false,recent_alert:false};
const mixed={...zero,product:{id:'mixed',name:'Monitor',brand:'Screens',model:'Gamma'},best_price:null,comparison_status:'incompatible_currencies',has_collection_error:false,recent_alert:false};
const data={products:[zero,missing,mixed],truncated:true,recent_alert_since:'2026-09-23T12:00:00Z'};

function load(path='../app/dashboard.tsx',extra={}) {
 const states=[];let cursor=0;
 const hooks={...React,useState(initial){const i=cursor++;if(!(i in states))states[i]=initial;return[states[i],v=>states[i]=v]}};
 const source=readFileSync(new URL(path,import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 const exports={},require=createRequire(import.meta.url);
 runInNewContext(compiled,{exports,require(name){if(name==='react')return hooks;if(name==='next/link')return {default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/dashboard'))return dashboard;if(name.endsWith('/prices'))return prices;if(name in extra)return extra[name];return require(name)}});
 return {exports,render(){cursor=0;return exports.default({data})}};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}

test('summary rendering preserves zero, identity, observation time, collection status and alert indicator',()=>{
 const h=load();const html=renderToStaticMarkup(h.exports.ProductCard({row:zero}));
 for(const text of ['Camera','Acme','Alpha','2 Listings','USD 0.00','2026-09-30T10:00:00Z','2026-09-30T12:00:00Z','Collection error','Triggered alert in the last 7 days','Observation freshness: 2 fresh'])assert.ok(html.includes(text),text);
 assert.match(html,/href="\/products\/p%20%2F%20zero"/);
 const absent=renderToStaticMarkup(h.exports.ProductCard({row:missing}));assert.match(absent,/Missing current offer/);assert.match(absent,/Latest valid observation: None/);assert.match(absent,/Latest collection attempt: Never/);assert.doesNotMatch(absent,/USD 0.00/);
 const incomparable=renderToStaticMarkup(h.exports.ProductCard({row:mixed}));assert.match(incomparable,/incompatible currencies/);assert.match(incomparable,/No comparable price/);assert.doesNotMatch(incomparable,/Best observed/);
 const stale=renderToStaticMarkup(h.exports.ProductCard({row:{...zero,best_price:null,comparison_status:'not_fresh',freshness:{stale:2}}}));assert.match(stale,/Observation freshness: 2 stale/);assert.match(stale,/Collection error/);assert.doesNotMatch(stale,/2 fresh/);
});

test('search matches name/brand/model case-insensitively and combines with every filter',()=>{
 for(const q of [' camERA ', 'acme','ALPHA'])assert.deepEqual(dashboard.filterProducts(data.products,q,'all'),[zero]);
 for(const [filter,expected] of [['priced',[zero,mixed]],['missing',[missing]],['error',[zero]],['alert',[zero]],['all',data.products]])assert.deepEqual(dashboard.filterProducts(data.products,'',filter),expected);
 assert.deepEqual(dashboard.filterProducts(data.products,'Keyboard','error'),[]);
});

test('dashboard controls apply search/filter and explain empty/bounded states',()=>{
 const h=load();let tree=h.render();
 nodes(tree).find(n=>n.type==='input').props.onChange({target:{value:'Keys'}});
 tree=h.render();assert.equal(nodes(tree).filter(n=>n.type===h.exports.ProductCard).length,1);
 nodes(tree).find(n=>n.type==='select').props.onChange({target:{value:'alert'}});
 assert.match(renderToStaticMarkup(h.render()),/No products match/);
 assert.match(renderToStaticMarkup(h.render()),/Search and filters apply only to these loaded products/);
 assert.match(renderToStaticMarkup(load().exports.default({data:{...data,products:[]}})),/No products yet/);
});

test('dashboard parser rejects malformed summaries and preserves explicit zero',()=>{
 assert.deepEqual(dashboard.parseDashboard(data),data);
 for(const row of [{...zero,best_price:{...zero.best_price,minor_units:NaN}},{...zero,comparison_status:'incompatible_currencies'},{...zero,latest_observation_at:'bad'},{...zero,has_collection_error:'false'}])assert.throws(()=>dashboard.parseDashboard({...data,products:[row]}));
});

test('home uses one summary request and lets errors reach existing error boundary',async()=>{
 let calls=0;const h=load('../app/page.tsx',{'../lib/api':{api:async(path,parse)=>{calls++;assert.equal(path,'/dashboard');return parse(data)}},'./dashboard':{default:()=>null}});
 await h.exports.default();assert.equal(calls,1);
 const bad=load('../app/page.tsx',{'../lib/api':{api:async()=>{throw new Error('unavailable')}},'./dashboard':{default:()=>null}});
 await assert.rejects(bad.exports.default(),/unavailable/);
 assert.match(renderToStaticMarkup(load('../app/loading.tsx').exports.default()),/Loading tracked prices/);
 let retried=false;const page=load('../app/error.tsx').exports.default({reset(){retried=true}});
 assert.match(renderToStaticMarkup(page),/role="alert"/);nodes(page).find(n=>n.type==='button').props.onClick();assert.equal(retried,true);
});
