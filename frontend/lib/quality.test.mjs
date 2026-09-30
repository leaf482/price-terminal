import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as quality from './quality.ts';
const record={id:'o',listing_id:'l',valid:true,invalidation_reason:'',invalidated_at:null,observation:{observed_at:'2026-01-01T00:00:00Z',source:'fixture',stock:'unknown',offer_price:0,currency:'USD'}};
function load(save=async()=>{}){
 const state=[],refs=[],requests=[];let cursor=0,refCursor=0,effect,reloads=0;
 const exports={},require=createRequire(import.meta.url),hooks={...React,useState(v){const i=cursor++;if(!(i in state))state[i]=v;return[state[i],v=>state[i]=typeof v==='function'?v(state[i]):v]},useRef(v){return refs[refCursor++]??=( {current:v} )},useEffect(f){effect=f}};
 const source=readFileSync(new URL('../app/products/[id]/quality-view.tsx',import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(compiled,{exports,Error,AbortController,setTimeout,clearTimeout,window:{location:{reload(){reloads++}}},FormData:class{constructor(form){this.form=form}get(){return this.form.reason}},require(name){if(name==='react')return hooks;if(name.endsWith('/quality'))return {...quality,invalidateObservation:save};if(name.endsWith('/api'))return {api:()=>new Promise((resolve,reject)=>requests.push({resolve,reject}))};return require(name)}});
 return {exports,requests,render(){cursor=refCursor=0;return exports.default({listingID:'l'})},setup(){return effect()},reloads:()=>reloads};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
async function settle(){for(let i=0;i<10;i++)await Promise.resolve()}
test('audit preserves facts and invalidation state',()=>{
 assert.deepEqual(quality.parseAudit(record),record);assert.equal(quality.parseAuditList({observations:[record],truncated:false}).observations[0].observation.offer_price,0);
 const h=load(),invalid={...record,valid:false,invalidation_reason:'wrong unit',invalidated_at:'2026-01-02T00:00:00Z'};
 const html=renderToStaticMarkup(h.exports.AuditRecord({record:invalid,busy:false,onInvalidate:async()=>{}}));assert.match(html,/wrong unit/);assert.match(html,/Invalidated/);assert.doesNotMatch(html,/<form/);assert.throws(()=>quality.parseAudit({...invalid,invalidated_at:null}));
});
test('invalidation validates reason and sends the exact ID',async()=>{
 const original=globalThis.fetch;let calls=0;globalThis.fetch=async(path,options)=>{calls++;assert.equal(path,'/api/observations/o/invalidate');assert.equal(JSON.parse(options.body).reason,'wrong price');return new Response('',{status:200})};
 try{await assert.rejects(quality.invalidateObservation('o',' '));assert.equal(calls,0);await quality.invalidateObservation('o','wrong price');assert.equal(calls,1);globalThis.fetch=async()=>new Response('',{status:404});await assert.rejects(quality.invalidateObservation('missing','wrong'),/not found/)}finally{globalThis.fetch=original}
});
test('audit action reloads all price/history state after successful invalidation',async()=>{
 let call;const h=load(async(...args)=>call=args);h.render();const cleanup=h.setup();try{h.requests[0].resolve({observations:[record],truncated:false});await settle();const view=nodes(h.render()).find(n=>n.props?.record);const detail=h.exports.AuditRecord(view.props);nodes(detail).find(n=>n.type==='form').props.onSubmit({preventDefault(){},currentTarget:{reason:'bad source'}});await settle();assert.deepEqual(call,['o','bad source']);assert.equal(h.reloads(),1)}finally{cleanup()}
});
test('failed invalidation keeps audit facts and reports error',async()=>{
 const h=load(async()=>{throw new Error('Unavailable')});h.render();const cleanup=h.setup();try{h.requests[0].resolve({observations:[record],truncated:false});await settle();await nodes(h.render()).find(n=>n.props?.record).props.onInvalidate('o','wrong');assert.equal(h.reloads(),0);assert.match(renderToStaticMarkup(h.render()),/Unavailable/)}finally{cleanup()}
});
