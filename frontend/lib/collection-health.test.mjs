import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as health from './collection-health.ts';
const row={listing_id:'constructor',product_id:'__proto__',product_name:'Camera',retailer_id:'toString',retailer_name:'Shop',tracking_enabled:true,observed_at:'2020-01-01T00:00:00Z',attempted_at:'2026-10-01T00:00:00Z',successful_at:'2026-09-01T00:00:00Z',outcome:'provider_error',error_summary:'provider_error'};
const never={...row,listing_id:'__proto__',product_name:'Apple',attempted_at:null,successful_at:null,observed_at:null,outcome:'',error_summary:''};
const disabled={...row,listing_id:'toString',tracking_enabled:false,outcome:'success',error_summary:''};
const data={listings:[row,never,disabled],truncated:true,counts:{total:3,healthy:0,error:1,never:1,disabled:1}};
test('health parser, filters, sorting and observation-only freshness',()=>{
 assert.deepEqual(health.parseCollectionHealth(data),data);assert.throws(()=>health.parseCollectionHealth({...data,listings:Array(101).fill(row)}));
 for(const [filter,want] of [['error',[row]],['never',[never]],['disabled',[disabled]],['enabled',[row,never]]])assert.deepEqual(health.collectionRows(data.listings,filter,'attempt'),want);
 assert.deepEqual(health.collectionRows(data.listings,'all','attempt'),[row,disabled,never]);
 assert.deepEqual(health.collectionRows(data.listings,'all','success'),[never,row,disabled]);
 assert.deepEqual(health.collectionRows(data.listings,'all','product'),[never,row,disabled]);
 const now=Date.parse(row.attempted_at);assert.equal(health.observationFreshness(row.observed_at,now),'stale');assert.equal(health.observationFreshness(null,now),'no observation');assert.equal(health.observationFreshness(row.attempted_at,now),'recent');assert.equal(health.observationFreshness(new Date(now-health.OBSERVATION_RECENT_MS).toISOString(),now),'recent');assert.equal(health.observationFreshness(row.attempted_at,now-1),'stale');
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('Collection page loads once, displays separate timestamps/context and supports empty/error states',async()=>{
 const states=[],requests=[];let cursor=0,effect;const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL('../app/collection/page.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
 runInNewContext(code,{exports,AbortController,setTimeout,clearTimeout,require(name){if(name==='react')return{...React,useState(v){const i=cursor++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=typeof v==='function'?v(states[i]):v]},useEffect(f){effect=f}};if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/collection-health'))return health;if(name.endsWith('/api'))return{api:path=>{assert.equal(path,'/collection/overview');return new Promise((resolve,reject)=>requests.push({resolve,reject}))}};return require(name)}});
 const render=()=>{cursor=0;return exports.default()};const settle=async()=>{for(let i=0;i<10;i++)await Promise.resolve()};
 assert.match(renderToStaticMarkup(render()),/Loading collection/);let cleanup=effect();requests[0].resolve(data);await settle();let tree=render();const html=renderToStaticMarkup(tree);
 for(const text of ['Camera','Shop','stale','no observation','provider_error','First 100',row.observed_at,row.attempted_at,row.successful_at])assert.ok(html.includes(text),text);assert.match(html,/href="\/listings\/constructor"/);assert.equal(requests.length,1);
 nodes(tree).find(n=>n.type==='select').props.onChange({target:{value:'disabled'}});assert.equal(nodes(render()).filter(n=>n.type==='tbody')[0].props.children.length,1);cleanup();
 cleanup=effect();requests[1].resolve({...data,listings:[]});await settle();assert.match(renderToStaticMarkup(render()),/No Listings match/);cleanup();
 cleanup=effect();requests[2].reject(new Error('timeout'));await settle();assert.match(renderToStaticMarkup(render()),/Could not load/);cleanup();
});
