import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as home from './home.ts';
import * as prices from './prices.ts';
import * as changes from './price-changes.ts';
const empty={counts:Object.fromEntries(Object.keys(home.countLabels).map(k=>[k,0])),price_changes:[],alert_events:[],collection_failures:[],errors:[]};
function load(file,api){const exports={},require=createRequire(import.meta.url);const code=ts.transpileModule(readFileSync(new URL(file,import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;runInNewContext(code,{exports,require(name){if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/api'))return{api};if(name.endsWith('/home'))return home;if(name.endsWith('/prices'))return prices;if(name.endsWith('/price-changes'))return changes;if(name==='./home-view')return load('../app/home-view.tsx');return require(name)}});return exports;}
test('Home empty, populated and partially failed sections preserve types and navigation',()=>{
 assert.deepEqual(home.parseHome(empty),empty);const view=load('../app/home-view.tsx').HomeView;
 const blank=renderToStaticMarkup(view({data:empty}));for(const text of ['No price changes','No triggered alerts','No recorded collection failures','Active Products','Archived Products'])assert.ok(blank.includes(text));
 const context={listing_id:'constructor',product_id:'__proto__',product_name:'Camera',retailer_id:'toString',retailer_name:'Shop'};
 const change={...context,previous:{id:'p',observed_at:'2026-01-01T00:00:00Z',minor_units:'100'},current:{id:'c',observed_at:'2026-01-02T00:00:00Z',minor_units:'0'},currency:'USD',change_minor:'-100',percentage:'-100.00',direction:'decreased'};
 const event={...context,alert_id:'a',observation_id:'o',kind:'target',minor_units:0,currency:'USD',triggered_at:'2026-01-03T00:00:00Z'};
 const failure={...context,id:'f',started_at:'2026-01-04T00:00:00Z',outcome:'provider_error',error_summary:'provider_error'};
 const populated={...empty,price_changes:[change],alert_events:[event],collection_failures:[failure]};assert.deepEqual(home.parseHome(populated),populated);
 const html=renderToStaticMarkup(view({data:populated}));for(const path of ['/products/__proto__','/listings/constructor','/alerts','/collection','/price-changes','/products'])assert.ok(html.includes(`href="${path}"`));for(const text of ['USD 0.00','Observed:','Triggered:','Attempt started:','provider_error'])assert.ok(html.includes(text));
 const partial={...populated,price_changes:null,errors:['price_changes']};assert.match(renderToStaticMarkup(view({data:home.parseHome(partial)})),/Price changes unavailable/);assert.match(renderToStaticMarkup(view({data:partial})),/Triggered:/);
 assert.throws(()=>home.parseHome({...populated,alert_events:Array(6).fill(event)}));
});
test('Home uses one overview request and retains useful navigation on total failure',async()=>{
 let calls=0;const page=load('../app/page.tsx',async(path,parse)=>{calls++;assert.equal(path,'/overview');return parse(empty)});assert.match(renderToStaticMarkup(await page.default()),/Tracker overview/);assert.equal(calls,1);
 const failed=load('../app/page.tsx',async()=>{throw new Error('offline')});const html=renderToStaticMarkup(await failed.default());assert.match(html,/Overview unavailable/);assert.match(html,/href="\/products"/);
});
