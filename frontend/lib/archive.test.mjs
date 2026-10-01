import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {setProductArchived} from './archive.ts';
import {parseProduct} from './api.ts';
test('archive requests carry only visibility state and report errors',async()=>{
 const saved=globalThis.fetch;
 try{for(const archived of [true,false]){globalThis.fetch=async(path,options)=>{assert.equal(path,'/api/products/p/archive');assert.equal(options.method,'PATCH');assert.deepEqual(JSON.parse(options.body),{archived});return new Response('{}')};await setProductArchived('p',archived)}
 globalThis.fetch=async()=>new Response('{}',{status:404});await assert.rejects(setProductArchived('p',true),/not found/);
 }finally{globalThis.fetch=saved}
 assert.equal(parseProduct({id:'p',name:'',brand:'',model:'',archived:true}).archived,true);
 assert.throws(()=>parseProduct({id:'p',name:'',brand:'',model:'',archived:'yes'}));
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('archive control preserves tracking guidance, toggles and refreshes only on success',async()=>{
 for(const archived of [true,false])for(const fail of [true,false]){
  const states=[],refs=[];let index=0,ri=0,refreshes=0;
  const exports={},require=createRequire(import.meta.url),hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]},useRef(v){return refs[ri++]??={current:v}}};
  const code=ts.transpileModule(readFileSync(new URL('../app/products/[id]/archive-controls.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
  runInNewContext(code,{exports,Error,require(name){if(name==='react')return hooks;if(name==='next/navigation')return{useRouter:()=>({refresh(){refreshes++}})};if(name.endsWith('/archive'))return{setProductArchived:async(id,value)=>{assert.equal(id,'p');assert.equal(value,!archived);if(fail)throw new Error('Failed')}};return require(name)}});
  const render=()=>{index=ri=0;return exports.default({id:'p',archived})};
  assert.match(renderToStaticMarkup(render()),archived?/Product: Archived/:/Product: Active/);
  assert.match(renderToStaticMarkup(render()),/tracking and collection continue unchanged/);
  await nodes(render()).find(n=>n.type==='button').props.onClick();assert.equal(refreshes,fail?0:1);if(fail)assert.match(renderToStaticMarkup(render()),/role="alert"/);
 }
});
