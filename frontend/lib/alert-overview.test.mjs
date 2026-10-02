import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import * as overview from './alert-overview.ts';
import * as prices from './prices.ts';
const context={product_id:'constructor',product_name:'Camera',retailer_id:'__proto__',retailer_name:'Shop'};
const target={...context,alert:{id:'constructor',listing_id:'toString',kind:'target',currency:'USD',threshold_minor_units:0,enabled:true,require_in_stock:true},last_triggered_at:'2026-10-01T00:00:00Z'};
const drop={...context,alert:{id:'__proto__',listing_id:'toString',kind:'drop',currency:'JPY',drop_basis_points:1000,enabled:false,require_in_stock:false},last_triggered_at:null};
const low={...context,alert:{id:'toString',listing_id:'toString',kind:'historical_low',currency:'USD',enabled:true,require_in_stock:false},last_triggered_at:null};
const event={...context,alert_id:'constructor',listing_id:'toString',observation_id:'__proto__',kind:'target',minor_units:0,currency:'USD',triggered_at:'2026-10-01T00:00:00Z'};
const data={alerts:[target,drop,low],events:[event],alerts_truncated:false,events_truncated:false};
test('overview parser and filters preserve all kinds, zero and unusual context IDs',()=>{
 assert.deepEqual(overview.parseAlertOverview(data),data);
 for(const [filter,expected] of [['all',[target,drop,low]],['enabled',[target,low]],['disabled',[drop]],['triggered',[target]],['never',[drop,low]]])assert.deepEqual(overview.filterAlerts(data.alerts,filter,'all'),expected);
 for(const row of data.alerts)assert.deepEqual(overview.filterAlerts(data.alerts,'all',row.alert.kind),[row]);
 assert.throws(()=>overview.parseAlertOverview({...data,alerts:Array(101).fill(target)}));assert.throws(()=>overview.parseAlertOverview({...data,events:Array(21).fill(event)}));
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('Alerts page loads once, displays context/events and reuses toggle API to update overview',async()=>{
 const states=[],refs=[];let index=0,ri=0,effect,reads=0,writes=0;
 const hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=typeof v==='function'?v(states[i]):v]},useRef(v){return refs[ri++]??={current:v}},useEffect(fn){effect=fn}};
 const exports={},require=createRequire(import.meta.url);const code=ts.transpileModule(readFileSync(new URL('../app/alerts/page.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
 runInNewContext(code,{exports,AbortController,setTimeout,clearTimeout,require(name){if(name==='react')return hooks;if(name==='next/link')return{default:({href,children})=>React.createElement('a',{href},children)};if(name.endsWith('/api'))return{api:async(path,parse)=>{reads++;assert.equal(path,'/alerts/overview');return parse(data)}};if(name.endsWith('/alerts'))return{saveAlert:async(listing,body,id)=>{writes++;assert.equal(listing,'toString');assert.equal(id,'constructor');assert.equal(body.enabled,false);return{...target.alert,enabled:false}}};if(name.endsWith('/prices'))return prices;if(name.endsWith('/alert-overview'))return overview;return require(name)}});
 const render=()=>{index=ri=0;return exports.default()};assert.match(renderToStaticMarkup(render()),/Loading alerts/);const cleanup=effect();
 try{for(let i=0;i<6;i++)await Promise.resolve();let tree=render();const html=renderToStaticMarkup(tree);assert.equal(reads,1);
 for(const text of ['Camera','Shop','constructor','__proto__','USD 0.00','10%','historical low','Last triggered: Never','Recent triggered events','Observed trigger value'])assert.ok(html.includes(text),text);assert.match(html,/href="\/listings\/toString"/);
 await nodes(tree).find(n=>n.type==='button'&&n.props.children==='Disable').props.onClick();assert.equal(writes,1);assert.equal(overview.filterAlerts(states[0].alerts,'disabled','all').length,2);
 nodes(render()).find(n=>n.type==='select').props.onChange({target:{value:'triggered'}});assert.equal(nodes(render()).filter(n=>n.type==='article').length,2);
 }finally{cleanup()}
});
