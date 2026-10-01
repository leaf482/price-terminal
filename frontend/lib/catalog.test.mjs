import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as catalog from './catalog.ts';

const product={id:'p',name:'Phone',brand:'Maker',model:'One'},retailer={id:'r',name:'Shop'};
const listing={id:'l',product_id:'p',retailer_id:'r',url:'https://example.com/item',retailer_product_id:''};
function harness(file,save=async()=>{}){
 const state=[],refs=[],requests=[];let cursor=0,refCursor=0,effect,refreshes=0;
 const hooks={...React,useState(v){const i=cursor++;if(!(i in state))state[i]=v;return[state[i],v=>{state[i]=typeof v==='function'?v(state[i]):v}]},useRef(v){const i=refCursor++;return refs[i]??=( {current:v} )},useEffect(f){effect=f}};
 const exports={},require=createRequire(import.meta.url);
 const source=readFileSync(new URL(file,import.meta.url),'utf8');
 const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(compiled,{exports,Error,AbortController,setTimeout,clearTimeout,FormData:class{constructor(form){this.fields=form.fields}get(key){return this.fields[key]??null}},require(name){
 if(name==='react')return hooks;if(name==='next/link')return {default:({children,...props})=>React.createElement('a',props,children)};
 if(name==='next/navigation')return {useRouter:()=>({refresh(){refreshes++}})};
 if(name==='./metadata-editor')return {default:()=>null};
 if(name.endsWith('/catalog'))return {...catalog,postCatalog:save};
 if(name.endsWith('/api'))return {api:(path,_parse,signal)=>new Promise((resolve,reject)=>{requests.push({path,resolve,reject});signal.addEventListener('abort',()=>reject(new Error('abort')),{once:true})})};return require(name)}});
 return {requests,refreshes:()=>refreshes,render(name,props){cursor=0;refCursor=0;return exports[name](props)},setup(){return effect()}};
}
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
async function settle(){for(let i=0;i<10;i++)await Promise.resolve()}
for(const [kind,fields] of [['products',product],['retailers',retailer],['listings',listing]]){
 test(`${kind} form success refreshes parent and preserves fields`,async()=>{
 const calls=[],created=[];let resets=0;
 const h=harness('../app/catalog/manager.tsx',async(...args)=>calls.push(args));
 const props={kind,products:[product],retailers:[retailer],onCreated:body=>created.push(body)};
 const form=h.render('CatalogForm',props);
 await form.props.onSubmit({preventDefault(){},currentTarget:{fields,reset(){resets++}}});
 assert.equal(calls[0][0],`/${kind}`);assert.deepEqual(calls[0][1],fields);assert.deepEqual(created,[fields]);assert.equal(resets,1);
 assert.match(renderToStaticMarkup(h.render('CatalogForm',props)),/Created/);
 });
 test(`${kind} form error preserves input and does not refresh`,async()=>{
 let creates=0;const h=harness('../app/catalog/manager.tsx',async()=>{throw new Error('This ID already exists.')});
 const props={kind,products:[],retailers:[],onCreated:()=>creates++};
 await h.render('CatalogForm',props).props.onSubmit({preventDefault(){},currentTarget:{fields,reset(){assert.fail('reset on failure')}}});
 assert.equal(creates,0);assert.match(renderToStaticMarkup(h.render('CatalogForm',props)),/role="alert"/);
 });
}
test('catalog optional retailer product ID and conflicts',async()=>{
 const form=new FormData();for(const [key,v] of Object.entries({...listing,retailer_product_id:'SKU-1'}))form.set(key,v);
 assert.equal(catalog.catalogBody('listings',form).retailer_product_id,'SKU-1');form.delete('retailer_product_id');assert.equal(catalog.catalogBody('listings',form).retailer_product_id,'');
 assert.deepEqual(catalog.parseListings([listing]),[listing]);assert.deepEqual(catalog.parseRetailers([retailer]),[retailer]);
 const original=globalThis.fetch;try{
 for(const [status,code,message] of [[409,'duplicate','already exists'],[400,'invalid','Invalid fields'],[404,'missing','not found'],[503,'collection_unavailable','No provider'],[409,'collection_busy','already running'],[502,'collection_failed','Previous observations']]){
 globalThis.fetch=async()=>new Response(JSON.stringify({error:{code,message:'secret'}}),{status});await assert.rejects(catalog.postCatalog('/listings',listing),new RegExp(message));
 }
 globalThis.fetch=async(path,options)=>{assert.equal(path,'/api/listings');assert.equal(options.method,'POST');assert.deepEqual(JSON.parse(options.body),listing);return new Response('',{status:201})};await catalog.postCatalog('/listings',listing);
 }finally{globalThis.fetch=original}
});
test('catalog creation callback reloads API data and navigational state',async()=>{
 const h=harness('../app/catalog/manager.tsx');h.render('default');const cleanup=h.setup();
 try{h.requests[0].resolve([product]);h.requests[1].resolve([retailer]);await settle();
 const tree=h.render('default');const form=nodes(tree).find(n=>n.props?.kind==='listings');form.props.onCreated(listing);
 assert.equal(h.refreshes(),1);cleanup();h.render('default');const second=h.setup();try{assert.equal(h.requests.length,4);h.requests[2].resolve([product]);h.requests[3].resolve([retailer]);await settle()}finally{second()}
 }finally{cleanup();await settle()}
});
test('manual refresh prevents concurrent clicks and reloads status after success or failure',async()=>{
 for(const fail of [false,true]){
 let resolve,calls=0;const waiting=new Promise(r=>resolve=r);
 const h=harness('../app/products/[id]/collection-controls.tsx',async path=>{calls++;assert.equal(path,'/listings/l/collect');await waiting;if(fail)throw new Error('Collection failed. Previous observations are preserved.')});
 const props={id:'l',status:{state:'success',last_attempted_at:'2026-01-02',last_successful_at:'2026-01-01'}};
 const button=nodes(h.render('default',props)).find(n=>n.type==='button'&&n.props.children==='Refresh price');const first=button.props.onClick();await button.props.onClick();assert.equal(calls,1);
 assert.ok(nodes(h.render('default',props)).find(n=>n.type==='button'&&n.props.children==='Collecting…').props.disabled);
 resolve();await first;assert.equal(h.refreshes(),1);const html=renderToStaticMarkup(h.render('default',props));assert.match(html,fail?/Previous observations/:/Collection succeeded/);assert.match(html,/Last successful: 2026-01-01/);
 }
});
test('inactive collection is honest and disables refresh',()=>{
 const h=harness('../app/products/[id]/collection-controls.tsx');const tree=h.render('default',{id:'l',status:{state:'inactive'}});assert.match(renderToStaticMarkup(tree),/No provider configured/);assert.ok(nodes(tree).find(n=>n.type==='button'&&n.props.children==='Refresh price').props.disabled);
});
