import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {csvHeader,importCSV} from './csv-import.ts';

test('CSV upload sends original bytes and import identity; row errors are retained',async()=>{
 const original=globalThis.fetch;
 const csv=csvHeader+'\n2026-01-02T03:04:05.123456789+05:30,USD,0,,,,,unknown\n';const file=new Blob([csv]);
 try{
  globalThis.fetch=async(path,options)=>{assert.equal(path,'/api/listings/l/observations/import');assert.equal(options.headers['X-Import-ID'],'id');assert.equal(await options.body.text(),csv);return new Response(JSON.stringify({data:{imported:1}}),{status:201})};
  assert.equal(await importCSV('l',file,'id'),1);
  globalThis.fetch=async()=>new Response(JSON.stringify({error:{rows:[{row:3,message:'invalid timestamp'},{row:5,message:'MSRP source required'}]}}),{status:400});
  await assert.rejects(importCSV('l',file,'id'),/Row 3: invalid timestamp\nRow 5: MSRP source required/);
  await assert.rejects(importCSV('l',new Blob(['x'.repeat(1048577)]),'id'),/1 MiB/);
 }finally{globalThis.fetch=original}
});
function load(save){
 const states=[],refs=[];let index=0,ri=0,refreshes=0;
 const exports={},require=createRequire(import.meta.url),hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]},useRef(v){return refs[ri++]??={current:v}}};
 const source=readFileSync(new URL('../app/products/[id]/csv-import.tsx',import.meta.url),'utf8');
 const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,Error,crypto:{randomUUID:()=> 'import-id'},require(name){if(name==='react')return hooks;if(name==='next/navigation')return {useRouter:()=>({refresh(){refreshes++}})};if(name.endsWith('/csv-import'))return {csvHeader,importCSV:save};return require(name)}});
 return {render(){index=ri=0;return exports.default({listingID:'l'})},refreshes:()=>refreshes};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('CSV control reports count and refreshes after success; errors keep the same retry ID',async()=>{
 for(const fail of [false,true]){
  const calls=[];const h=load(async(...args)=>{calls.push(args);if(fail)throw new Error('Row 3: invalid timestamp');return 2});const file=new Blob(['csv']);
  nodes(h.render()).find(n=>n.type==='input').props.onChange({target:{files:[file]}});
  const submit=()=>nodes(h.render()).find(n=>n.type==='form').props.onSubmit({preventDefault(){}});
  await submit();assert.deepEqual(calls[0],['l',file,'import-id']);const html=renderToStaticMarkup(h.render());
  if(fail){assert.match(html,/Row 3: invalid timestamp/);assert.equal(h.refreshes(),0);await submit();assert.equal(calls[1][2],calls[0][2]);}
  else{assert.match(html,/Imported 2 rows/);assert.equal(h.refreshes(),1);assert.ok(nodes(h.render()).find(n=>n.type==='button').props.disabled);}
 }
});
