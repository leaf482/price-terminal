import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {setTracking} from './tracking.ts';

test('tracking request uses explicit booleans and reports failures',async()=>{
 const old=globalThis.fetch;try{
  for(const enabled of [false,true]){globalThis.fetch=async(path,options)=>{assert.equal(path,'/api/listings/l/tracking');assert.equal(options.method,'PATCH');assert.equal(JSON.parse(options.body).tracking_enabled,enabled);return new Response('{}',{status:200})};await setTracking('l',enabled)}
  globalThis.fetch=async()=>new Response('',{status:500});await assert.rejects(setTracking('l',false),/Unable to change/);
 }finally{globalThis.fetch=old}
});
function load(file,save=async()=>{}){
 let index=0,refIndex=0,reloads=0;const states=[],refs=[],exports={},require=createRequire(import.meta.url);
 const hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],x=>states[i]=x]},useRef(v){return refs[refIndex++]??={current:v}}};
 const source=readFileSync(new URL('../app/products/[id]/'+file,import.meta.url),'utf8');
 const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,Error,window:{location:{reload(){reloads++}}},require(name){if(name==='react')return hooks;if(name==='next/navigation')return {useRouter:()=>({refresh(){}})};if(name.endsWith('/tracking'))return {setTracking:save};if(name.endsWith('/catalog'))return {postCatalog:save};return require(name)}});
 return {render(props){index=refIndex=0;return exports.default(props)},reloads:()=>reloads};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('tracking toggle displays state, saves the inverse and refreshes without hiding history',async()=>{
 for(const enabled of [false,true]){
  let call;const h=load('tracking-controls.tsx',async(...args)=>call=args);const tree=h.render({id:'l',enabled});
  assert.match(renderToStaticMarkup(tree),enabled?/Tracking enabled/:/Tracking disabled/);assert.match(renderToStaticMarkup(tree),/Record price entries are still allowed/);
  await nodes(tree).find(n=>n.type==='button').props.onClick();assert.deepEqual(call,['l',!enabled]);assert.equal(h.reloads(),1);
 }
 const h=load('tracking-controls.tsx',async()=>{throw new Error('Failed')});await nodes(h.render({id:'l',enabled:true})).find(n=>n.type==='button').props.onClick();assert.equal(h.reloads(),0);assert.match(renderToStaticMarkup(h.render({id:'l',enabled:true})),/role="alert"/);
});
test('disabled tracking prevents refresh without disguising observation freshness',async()=>{
 let calls=0;const h=load('collection-controls.tsx',async()=>calls++);const tree=h.render({id:'l',trackingEnabled:false,status:{state:'disabled',last_successful_at:'2026-01-01T00:00:00Z'}});
 const button=nodes(tree).find(n=>n.type==='button'&&n.props.children==='Refresh price');assert.equal(button.props.disabled,true);await button.props.onClick();assert.equal(calls,0);
 assert.match(renderToStaticMarkup(tree),/Status: disabled/);assert.match(renderToStaticMarkup(tree),/2026-01-01/);
 const active=h.render({id:'l',trackingEnabled:true,status:{state:'success'}});assert.equal(nodes(active).find(n=>n.type==='button'&&n.props.children==='Refresh price').props.disabled,false);
});
