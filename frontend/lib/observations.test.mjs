import { test } from 'node:test';
import assert from 'node:assert/strict';
import { observationBody, recordObservation } from './observations.ts';
import { backendURL } from './backend-url.ts';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

function form(extra = {}) {
 const f = new FormData();
 for (const [k,v] of Object.entries({ observed_at: '2026-09-30T12:00:00Z', source: 'receipt', stock: 'unknown', currency: '', ...extra })) f.set(k,v);
 return f;
}
test('manual prices preserve missing, zero, currencies and explicit evidence',()=>{
 const stock = observationBody(form(), 'id');assert.equal(stock.offer_price,undefined);assert.equal(stock.currency,undefined);
 assert.deepEqual(observationBody(form({currency:'USD',offer_price:'0',sale_price:'1.25'}),'id').offer_price,{minor_units:0,currency:'USD'});
 assert.equal(observationBody(form({currency:'USD',sale_price:'1.25'}),'id').sale_price.minor_units,125);
 assert.equal(observationBody(form({currency:'JPY',offer_price:'100'}),'id').offer_price.minor_units,100);
 for (const extra of [{offer_price:'1'},{currency:'JPY',offer_price:'1.1'},{currency:'USD',offer_price:'-1'},{currency:'USD',msrp:'1'},{observed_at:'2026-09-30T12:00:00'},{source:''}]) assert.throws(()=>observationBody(form(extra),'id'));
 assert.equal(observationBody(form({currency:'USD',msrp:'0',msrp_source:'manufacturer'}),'id').msrp.minor_units,0);
});
test('manual request preserves identity and reports backend errors',async()=>{
 const original=globalThis.fetch;let call;
 try {globalThis.fetch=async(path,options)=>{call={path,options};return new Response('{}',{status:201})};
 const body=observationBody(form(),'retry-id');await recordObservation('listing / one',body);
 assert.equal(call.path,'/api/listings/listing%20%2F%20one/observations');assert.deepEqual(JSON.parse(call.options.body),body);
 globalThis.fetch=async()=>new Response(JSON.stringify({error:{message:'Already recorded'}}),{status:409});await assert.rejects(recordObservation('l',body),/Already recorded/);
 }finally{globalThis.fetch=original}
});
test('backend origin defaults and errors',()=>{
 assert.equal(backendURL(undefined),'http://127.0.0.1:8080');assert.equal(backendURL('https://backend.example/'),'https://backend.example');
 for(const value of ['', 'invalid','ftp://example.com','http://user:secret@example.com','http://example.com/path','http://example.com/?q=1'])assert.throws(()=>backendURL(value),/Invalid BACKEND_URL/);
});

test('manual timestamp text reaches the request unchanged; backend errors are surfaced',async()=>{
 const original=globalThis.fetch;
 try {
  for(const input of ['2026-02-30T12:00:00Z','2026-01-02T03:04:05.123456789Z','2026-01-02T03:04:05.123456789+05:30','  2026-01-02T03:04:05-07:00 \n']) {
   const invalid=input.startsWith('2026-02-30');let calls=0;
   globalThis.fetch=async(_path,options)=>{
    calls++;assert.equal(JSON.parse(options.body).observed_at,input.trim());
    return new Response(JSON.stringify(invalid ? {error:{message:'expected manual observation JSON'}} : {}),{status:invalid ? 400 : 201});
   };
   const request=recordObservation('l',observationBody(form({observed_at:input}),'id'));
   if(invalid)await assert.rejects(request,/expected manual observation JSON/);else await request;
   assert.equal(calls,1);
  }
 }finally{globalThis.fetch=original}
 for(const input of ['', '  '])assert.throws(()=>observationBody(form({observed_at:input}),'id'),/Observation time/);
 const missing=form();missing.delete('observed_at');assert.throws(()=>observationBody(missing,'id'),/Observation time/);
});

function harness(save) {
 const state=[],refs=[];let cursor=0,refCursor=0,reloads=0;
 const hooks={...React,useState(v){const i=cursor++;if(!(i in state))state[i]=v;return[state[i],v=>state[i]=v]},useRef(v){return refs[refCursor++]??={current:v}}};
 const source=readFileSync(new URL('../app/products/[id]/record-price.tsx',import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 const exports={},require=createRequire(import.meta.url);
 runInNewContext(compiled,{exports,Error,Date:class extends Date{constructor(){super('2026-01-02T03:04:05.123Z')}},crypto:{randomUUID:()=> 'stable-id'},FormData:class{constructor(f){return f}},window:{location:{reload(){reloads++}}},require(name){if(name==='react')return hooks;if(name.endsWith('/observations'))return {observationBody,recordObservation:save};return require(name)}});
 return {render(){cursor=refCursor=0;return exports.default({listingID:'l'})},reloads:()=>reloads};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('opening entry supplies a default but preserves an edited timestamp',()=>{
 const h=harness(async()=>{});
 const input=()=>nodes(h.render()).find(n=>n.props?.name==='observed_at');
 assert.equal(input().props.value,'');
 h.render().props.onToggle({currentTarget:{open:true}});
 assert.equal(input().props.value,'2026-01-02T03:04:05.123Z');
 assert.equal(observationBody(form({observed_at:input().props.value}),'id').observed_at,'2026-01-02T03:04:05.123Z');
 const entered='2026-01-02T03:04:05.123456789+05:30';
 input().props.onChange({target:{value:entered}});
 h.render().props.onToggle({currentTarget:{open:false}});
 h.render().props.onToggle({currentTarget:{open:true}});
 assert.equal(input().props.value,entered);
});
test('entry success reloads current/history/events and blocks repeated submissions',async()=>{
 let resolve,calls=0;const h=harness(async(id,body)=>{calls++;assert.equal(id,'l');assert.equal(body.offer_price.minor_units,0);return new Promise(r=>resolve=r)});
 const submit=nodes(h.render()).find(n=>n.type==='form').props.onSubmit;
 const event={preventDefault(){},currentTarget:form({currency:'USD',offer_price:'0'})};
 const pending=submit(event);await submit(event);assert.equal(calls,1);assert.equal(h.reloads(),0);
 resolve();await pending;assert.equal(h.reloads(),1);
});
test('entry error stays visible and retry retains ID without reloading',async()=>{
 const ids=[];const h=harness(async(_,body)=>{ids.push(body.id);throw new Error('Database unavailable')});
 const submit=()=>nodes(h.render()).find(n=>n.type==='form').props.onSubmit({preventDefault(){},currentTarget:form()});
 await submit();assert.match(renderToStaticMarkup(h.render()),/Database unavailable/);await submit();
 assert.deepEqual(ids,['stable-id','stable-id']);assert.equal(h.reloads(),0);
});
