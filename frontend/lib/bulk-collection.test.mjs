import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {refreshSelected} from './bulk-collection.ts';
import * as health from './collection-health.ts';

test('bulk client submits explicit IDs only and reports transport errors without retry',async()=>{
 const original=globalThis.fetch;let calls=0;
 try{globalThis.fetch=async(path,options)=>{calls++;assert.equal(path,'/api/collection/refresh');assert.deepEqual(JSON.parse(options.body),{listing_ids:['constructor','__proto__']});return Response.json({data:[{listing_id:'constructor',outcome:'success',observation_id:'o'},{listing_id:'__proto__',outcome:'unavailable',error_summary:'tracking_disabled'}]})};
 await assert.rejects(refreshSelected([]));await assert.rejects(refreshSelected(Array(21).fill('a')));assert.equal(calls,0);
 assert.equal((await refreshSelected(['constructor','__proto__'])).length,2);assert.equal(calls,1);
 globalThis.fetch=async()=>new Response('',{status:502});await assert.rejects(refreshSelected(['a']),/some attempts may have completed/);
 }finally{globalThis.fetch=original}
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('Collection selection, duplicate-submit guard, results and health reload',async()=>{
 const states=[],refs=[];let cursor=0,ri=0,effect,reads=0,writes=0,resolve;
 const listing={listing_id:'constructor',product_id:'p',product_name:'P',retailer_id:'r',retailer_name:'R',tracking_enabled:false,observed_at:null,attempted_at:null,successful_at:null,outcome:'',error_summary:''};
 const data={listings:[listing],truncated:false,counts:{total:1,healthy:0,error:0,never:0,disabled:1}};
 const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL('../app/collection/page.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
 runInNewContext(code,{exports,AbortController,setTimeout,clearTimeout,require(name){
 if(name==='react')return{...React,useState(v){const i=cursor++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=typeof v==='function'?v(states[i]):v]},useRef(v){return refs[ri++]??={current:v}},useEffect(f){effect=f}};
 if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};
 if(name.endsWith('/collection-health'))return health;
 if(name.endsWith('/api'))return{api:async()=>{reads++;return data}};
 if(name.endsWith('/bulk-collection'))return{refreshSelected:ids=>{writes++;assert.deepEqual(Array.from(ids),['constructor']);return new Promise(r=>resolve=r)}};
 return require(name)}});
 const render=()=>{cursor=ri=0;return exports.default()};const settle=async()=>{for(let i=0;i<8;i++)await Promise.resolve()};
 render();let cleanup=effect();await settle();let tree=render();const checkbox=nodes(tree).find(n=>n.type==='input');assert.equal(checkbox.props.disabled,false);checkbox.props.onChange({target:{checked:true}});
 tree=render();const button=nodes(tree).find(n=>n.type==='button'&&n.props.children==='Refresh selected');const pending=button.props.onClick();button.props.onClick();assert.equal(writes,1);assert.ok(nodes(render()).find(n=>n.type==='input').props.disabled);
 resolve([{listing_id:'constructor',outcome:'unavailable',error_summary:'tracking_disabled'}]);await pending;
 assert.match(renderToStaticMarkup(render()),/constructor: unavailable/);assert.match(renderToStaticMarkup(render()),/tracking_disabled/);assert.equal(nodes(render()).find(n=>n.type==='input').props.disabled,false);
 cleanup();cleanup=effect();await settle();assert.equal(reads,2);cleanup();
});
