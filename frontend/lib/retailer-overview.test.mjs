import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as overview from './retailer-overview.ts';
import * as prices from './prices.ts';
const a={product:{id:'constructor',name:'Camera',brand:'Acme',model:'Alpha'},current:{listing:{id:'constructor',product_id:'constructor',retailer_id:'constructor',url:'https://example.com/a',tracking_enabled:true},observation:{observed_at:'2026-01-01T00:00:00Z',source:'fixture',stock:'in_stock',offer_price:0,currency:'USD'},freshness:'stale',collection:{state:'failed',error:'collection_failed',last_attempted_at:'2026-10-01T00:00:00Z'}}};
const b={product:{id:'b',name:'Keyboard',brand:'Keys',model:'Beta'},current:{listing:{id:'b',product_id:'b',retailer_id:'constructor',url:'https://example.com/b',tracking_enabled:false},observation:null,freshness:'missing',collection:{state:'disabled'}}};
const data={retailer:{id:'constructor',name:'Real shop'},listings:[a,b],truncated:true};
test('Retailer parser and filters preserve zero, missing, metadata and prototype-like IDs',()=>{
 assert.deepEqual(overview.parseRetailerOverview(data),data);
 for(const q of ['camera','ACME','Alpha'])assert.deepEqual(overview.filterRetailerListings(data.listings,q,'all'),[a]);
 for(const [filter,want] of [['enabled',[a]],['disabled',[b]],['priced',[a]],['error',[a]],['all',[a,b]]])assert.deepEqual(overview.filterRetailerListings(data.listings,'',filter),want);
 assert.deepEqual(overview.filterRetailerListings(data.listings,'Keys','enabled'),[]);
 assert.throws(()=>overview.parseRetailerOverview({...data,listings:Array(101).fill(a)}));
 assert.throws(()=>overview.parseRetailerOverview({...data,retailer:{id:'constructor',name:12}}));
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
function load(file,extra={}){
 const states=[];let index=0;const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL(file,import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,require(name){if(name==='react')return{...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]}};if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/retailer-overview'))return overview;if(name.endsWith('/prices'))return prices;if(Object.hasOwn(extra,name))return extra[name];return require(name)}});
 return{exports,render(d=data){index=0;return exports.default({data:d})}};
}
test('Retailer overview renders independent prices/status and combined filters',()=>{
 const h=load('../app/retailers/[id]/overview.tsx');let tree=h.render();const html=renderToStaticMarkup(tree);
 for(const text of ['Camera','Acme','Alpha','USD 0.00','Price unavailable','Tracking: disabled','Collection: failed','2026-01-01T00:00:00Z','2026-10-01T00:00:00Z','stale','first 100','2 of 2'])assert.ok(html.includes(text),text);
 assert.match(html,/href="\/products\/constructor#constructor"/);
 nodes(tree).find(n=>n.type==='input').props.onChange({target:{value:'Beta'}});tree=h.render();assert.equal(nodes(tree).filter(n=>n.type==='article').length,1);
 nodes(tree).find(n=>n.type==='select').props.onChange({target:{value:'priced'}});assert.match(renderToStaticMarkup(h.render()),/No Listings match/);
 assert.match(renderToStaticMarkup(h.render({...data,listings:[]})),/No Listings for this Retailer/);
});
test('Retailer page uses one overview request and exposes loading/error states',async()=>{
 let calls=0;
 const h=load('../app/retailers/[id]/page.tsx',{'../../../lib/api':{api:async(path,parse)=>{calls++;assert.equal(path,'/retailers/constructor/overview');return parse(data)}},'./overview':{default:()=>null}});
 const page=await h.exports.default({params:Promise.resolve({id:'constructor'})});assert.equal(calls,1);assert.match(renderToStaticMarkup(page),/Real shop/);
 const bad=load('../app/retailers/[id]/page.tsx',{'../../../lib/api':{api:async()=>{throw new Error('not found')}},'./overview':{default:()=>null}});
 await assert.rejects(bad.exports.default({params:Promise.resolve({id:'missing'})}),/not found/);
 assert.match(renderToStaticMarkup(load('../app/retailers/[id]/loading.tsx').exports.default()),/Loading Retailer Listings/);
 assert.match(renderToStaticMarkup(load('../app/retailers/[id]/error.tsx').exports.default({reset(){}})),/role="alert"/);
});
