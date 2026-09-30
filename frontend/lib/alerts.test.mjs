import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as alerts from './alerts.ts';
import * as prices from './prices.ts';

const alert={id:'a',listing_id:'l',kind:'target',currency:'USD',threshold_minor_units:0,enabled:true,require_in_stock:true};
const event={alert_id:'a',listing_id:'l',observation_id:'o',kind:'target',triggered_at:'2026-09-29T01:00:00Z',observed_at:'2026-09-29T00:00:00Z',minor_units:0,currency:'USD',price_basis:'offer_price',comparison_minor_units:null};
test('exact alert inputs and response parsing preserve zero and currency precision',()=>{
 assert.equal(alerts.alertBody('target','USD','0',true).threshold_minor_units,0);
 assert.equal(alerts.alertBody('target','USD','12.34',true).threshold_minor_units,1234);
 assert.equal(alerts.alertBody('target','JPY','1234',true).threshold_minor_units,1234);
 assert.equal(alerts.alertBody('drop','JPY','12.34',false).drop_basis_points,1234);
 assert.equal(alerts.alertBody('historical_low','JPY','',true).threshold_minor_units,undefined);
 for(const [kind,currency,input] of [['target','JPY','1.1'],['target','USD','-1'],['target','USD','9007199254740992'],['drop','USD','0'],['drop','USD','100.01']])assert.throws(()=>alerts.alertBody(kind,currency,input,true));
 assert.deepEqual(alerts.parseAlerts([alert]),[alert]);
 for(const threshold of [undefined,null,-1,Number.MAX_SAFE_INTEGER+1])assert.throws(()=>alerts.parseAlert({...alert,threshold_minor_units:threshold}));
 assert.deepEqual(alerts.parseEvents({events:[event],truncated:false}).events,[event]);
});
test('create and disable requests use listing endpoints and explicit boolean',async()=>{
 const original=globalThis.fetch,calls=[];
 globalThis.fetch=async(path,options)=>{calls.push({path,...options});return new Response(JSON.stringify({data:{...alert,enabled:options.method==='POST'}}),{status:200})};
 try{
 await alerts.saveAlert('l',alerts.alertBody('target','USD','0',true));
 const result=await alerts.saveAlert('l',{enabled:false},'a');
 assert.equal(result.enabled,false);assert.equal(calls[0].method,'POST');assert.equal(calls[0].path,'/api/listings/l/alerts');assert.equal(JSON.parse(calls[0].body).threshold_minor_units,0);
 assert.equal(calls[1].method,'PATCH');assert.equal(calls[1].path,'/api/listings/l/alerts/a');assert.deepEqual(JSON.parse(calls[1].body),{enabled:false});
 globalThis.fetch=async()=>new Response('',{status:500});await assert.rejects(alerts.saveAlert('l',{}),/500/);
 }finally{globalThis.fetch=original}
});

function harness(){
 const states=[],requests=[],saves=[];let cursor=0,effect;
 const hooks={...React,useState(initial){const i=cursor++;if(!(i in states))states[i]=initial;return[states[i],v=>{states[i]=typeof v==='function'?v(states[i]):v}]},useEffect(f){effect=f}};
 const source=readFileSync(new URL('../app/products/[id]/alert-view.tsx',import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 const require=createRequire(import.meta.url),exports={};
 runInNewContext(compiled,{exports,AbortController,setTimeout,clearTimeout,require(name){
 if(name==='react')return hooks;if(name==='../../../lib/prices')return prices;
 if(name==='../../../lib/alerts')return {...alerts,saveAlert:async(...args)=>{saves.push(args);return alert}};
 if(name==='../../../lib/api')return {api:(path,_parse,signal)=>new Promise((resolve,reject)=>{requests.push({path,resolve,reject});signal.addEventListener('abort',()=>reject(new Error('aborted')),{once:true})})};return require(name)}});
 return {requests,saves,render(){cursor=0;return exports.default({listingID:'l'})},html(){return renderToStaticMarkup(this.render())},setup(){return effect()}};
}
function nodes(element){if(!element||typeof element!=='object')return[];if(Array.isArray(element))return element.flatMap(nodes);return[element,...nodes(element.props?.children)]}
async function settle(){for(let i=0;i<12;i++)await Promise.resolve()}
test('alert form creation, disable control and triggered event display',async()=>{
 const h=harness();assert.match(h.html(),/Loading alerts/);const cleanup=h.setup();
 try{
 h.requests[0].resolve([alert]);h.requests[1].resolve({events:[event],truncated:false});await settle();
 assert.match(h.html(),/USD 0.00/);assert.match(h.html(),/Observation: o/);assert.match(h.html(),/Promotions and EffectivePrice are excluded/);
 let elements=nodes(h.render());elements.find(n=>n.type==='input'&&n.props.inputMode==='decimal').props.onChange({target:{value:'0'}});
 elements=nodes(h.render());elements.find(n=>n.type==='form').props.onSubmit({preventDefault(){}});await settle();
 assert.equal(h.saves[0][1].threshold_minor_units,0);
 elements=nodes(h.render());elements.find(n=>n.type==='button'&&n.props.children==='Disable').props.onClick();await settle();
 assert.equal(h.saves[1][1].enabled,false);assert.equal(h.saves[1][2],'a');
 cleanup();const refreshedCleanup=h.setup();try{
 h.requests[2].resolve([{...alert,enabled:false}]);h.requests[3].resolve({events:[event],truncated:false});await settle();
 assert.match(h.html(),/Disabled/);assert.ok(nodes(h.render()).some(n=>n.type==='button'&&n.props.children==='Enable'));
 }finally{refreshedCleanup()}
 }finally{cleanup();await settle()}
});
test('cleanup ignores obsolete errors and replacement loads empty state',async()=>{
 const h=harness();h.render();h.setup()();const cleanup=h.setup();
 try{await settle();h.requests[2].resolve([]);h.requests[3].resolve({events:[],truncated:false});await settle();assert.match(h.html(),/No alerts configured/);assert.match(h.html(),/No triggered events yet/);assert.doesNotMatch(h.html(),/role="alert"/)}finally{cleanup();await settle()}
});
test('genuine alert loading failure is visible',async()=>{
 const h=harness();h.render();const cleanup=h.setup();try{h.requests[0].reject(new Error('unavailable'));await settle();assert.match(h.html(),/Could not load alerts\/events/)}finally{cleanup();await settle()}
});
