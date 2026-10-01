import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { downloadObservationCSV } from './csv-export.ts';

test('export downloads exact backend bytes and reports errors without downloading', async () => {
 const saved={fetch:globalThis.fetch,document:globalThis.document,create:URL.createObjectURL,revoke:URL.revokeObjectURL,setTimeout:globalThis.setTimeout};
 let downloaded,clicked=0,removed=0,revoked=0;
 const link={click(){clicked++},remove(){removed++}};
 try {
  globalThis.document={createElement:()=>link,body:{appendChild(node){assert.equal(node,link)}}};
  URL.createObjectURL=blob=>{downloaded=blob;return 'blob:csv'};URL.revokeObjectURL=url=>{assert.equal(url,'blob:csv');revoked++};
  globalThis.setTimeout=fn=>{fn();return 0};
  const csv='observation_id,source\na,"quotes, and text"\n';
  globalThis.fetch=async path=>{assert.equal(path,'/api/listings/a%2Fb/observations/export');return new Response(csv)};
  await downloadObservationCSV('a/b');assert.equal(await downloaded.text(),csv);assert.equal(link.download,'listing-observations.csv');assert.equal(clicked,1);assert.equal(removed,1);assert.equal(revoked,1);
  globalThis.fetch=async()=>new Response(JSON.stringify({error:{message:'Export exceeds the 10000 observation limit'}}),{status:422});
  await assert.rejects(downloadObservationCSV('l'),/10000/);assert.equal(clicked,1);
  globalThis.fetch=async()=>{throw new Error('Network unavailable')};
  await assert.rejects(downloadObservationCSV('l'),/Network unavailable/);
 } finally {globalThis.fetch=saved.fetch;globalThis.document=saved.document;URL.createObjectURL=saved.create;URL.revokeObjectURL=saved.revoke;globalThis.setTimeout=saved.setTimeout}
});

test('export control disables pending action and renders failure then recovers', async () => {
 const states=[],refs=[];let index=0,ri=0,resolve,reject;
 const exports={},require=createRequire(import.meta.url);
 const hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]},useRef(v){return refs[ri++]??={current:v}}};
 const source=readFileSync(new URL('../app/products/[id]/csv-export.tsx',import.meta.url),'utf8');
 const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,Error,require(name){if(name==='react')return hooks;if(name.endsWith('/csv-export'))return{downloadObservationCSV(id){assert.equal(id,'l');return new Promise((yes,no)=>{resolve=yes;reject=no})}};return require(name)}});
 const render=()=>{index=ri=0;return exports.default({listingID:'l'})};
 const button=()=>render().props.children[0];
 const first=button().props.onClick();assert.equal(button().props.disabled,true);reject(new Error('Export limit reached'));await first;
 assert.match(renderToStaticMarkup(render()),/role="alert".*Export limit reached/);
 const second=button().props.onClick();resolve();await second;
 assert.equal(button().props.disabled,false);assert.doesNotMatch(renderToStaticMarkup(render()),/role="alert"/);
});
