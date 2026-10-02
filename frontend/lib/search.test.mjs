import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {parseSearch,searchCatalog} from './search.ts';
const data={products:[{id:'constructor',name:'Camera',brand:'Brand',model:'Model',archived:true}],retailers:[{id:'__proto__',name:'Shop'}],listings:[{id:'toString',product_id:'constructor',retailer_id:'__proto__',url:'https://example.com/',retailer_product_id:'SKU',tracking_enabled:false}]};
test('search parser preserves typed groups and unusual IDs; query text is encoded literally',async()=>{
 assert.deepEqual(parseSearch(data),data);assert.throws(()=>parseSearch({...data,products:Array(21).fill(data.products[0])}));
 const saved=globalThis.fetch;let calls=0;
 try{globalThis.fetch=async url=>{calls++;assert.equal(new URL(url).searchParams.get('q'),"https://a/?q=%_'\\");return Response.json({data})};assert.deepEqual(await searchCatalog('   '),{products:[],retailers:[],listings:[]});assert.equal(calls,0);assert.deepEqual(await searchCatalog("https://a/?q=%_'\\"),data)}finally{globalThis.fetch=saved}
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
function load(search){
 const states=[],refs=[];let index=0,ri=0;const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL('../app/global-search.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,AbortController,setTimeout,clearTimeout,require(name){if(name==='react')return{...React,useEffect(){},useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]},useRef(v){return refs[ri++]??={current:v}}};if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/search'))return{searchCatalog:search};return require(name)}});
 const render=()=>{index=ri=0;return exports.default()};return{render,type(q){nodes(render()).find(n=>n.type==='input').props.onChange({target:{value:q}})},submit(){return nodes(render()).find(n=>n.type==='form').props.onSubmit({preventDefault(){}})}};
}
test('global search submits explicitly, groups links, labels inactive state, and handles loading/empty/error',async()=>{
 let calls=0,resolve;const h=load(async()=>{calls++;return new Promise(r=>resolve=r)});
 await h.submit();assert.equal(calls,0);assert.match(renderToStaticMarkup(h.render()),/Enter a search query/);
 h.type('cam');h.type('camera');assert.equal(calls,0);const pending=h.submit();assert.match(renderToStaticMarkup(h.render()),/Searching/);resolve(data);await pending;
 const html=renderToStaticMarkup(h.render());for(const text of ['Products','Retailers','Listings','Archived','Tracking disabled','Brand','Model','SKU'])assert.ok(html.includes(text));
 for(const path of ['/products/constructor','/retailers/__proto__','/listings/toString'])assert.ok(html.includes(`href="${path}"`));
 const empty=load(async()=>({products:[],retailers:[],listings:[]}));empty.type('absent');await empty.submit();assert.match(renderToStaticMarkup(empty.render()),/No matching catalog records/);
 const bad=load(async()=>{throw new Error('failure')});bad.type('x');await bad.submit();assert.match(renderToStaticMarkup(bad.render()),/Search failed/);
});
test('editing query ignores an obsolete successful request',async()=>{
 let resolve;const h=load(()=>new Promise(r=>resolve=r));h.type('old');const pending=h.submit();h.type('new');resolve(data);await pending;assert.doesNotMatch(renderToStaticMarkup(h.render()),/Camera/);
});
