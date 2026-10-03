import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {parseAttempts} from './collection-attempts.ts';
const attempt={id:'constructor',listing_id:'__proto__',trigger:'manual',started_at:'2026-01-01T00:00:00Z',finished_at:'2026-01-01T00:00:01Z',outcome:'success',observation_id:'toString'};
test('attempt parser preserves references and bounds',()=>{
 assert.deepEqual(parseAttempts([attempt]),[attempt]);assert.throws(()=>parseAttempts(Array(51).fill(attempt)));assert.throws(()=>parseAttempts([{...attempt,outcome:'secret'}]));
});
test('attempt history renders loading, safe outcomes, references, empty and errors; ignores obsolete requests',async()=>{
 const state=[],requests=[];let cursor=0,effect;const exports={},require=createRequire(import.meta.url);
 const code=ts.transpileModule(readFileSync(new URL('../app/listings/[id]/attempt-history.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
 runInNewContext(code,{exports,AbortController,setTimeout,clearTimeout,require(name){if(name==='react')return{...React,useState(v){const i=cursor++;if(!(i in state))state[i]=v;return[state[i],v=>state[i]=v]},useEffect(f){effect=f}};if(name.endsWith('/api'))return{api:(path)=>{assert.equal(path,'/listings/__proto__/collection-attempts');return new Promise((resolve,reject)=>requests.push({resolve,reject}))}};if(name.endsWith('/collection-attempts'))return{parseAttempts};return require(name)}});
 const render=()=>{cursor=0;return renderToStaticMarkup(exports.default({listingID:'__proto__'}))};const settle=async()=>{for(let i=0;i<5;i++)await Promise.resolve()};
 assert.match(render(),/Loading/);const cleanup=effect();cleanup();effect();requests[0].reject(new Error('aborted'));requests[1].resolve([attempt,{...attempt,id:'failure',trigger:'scheduled',outcome:'provider_error',observation_id:undefined,error_summary:'provider_error'}]);await settle();
 const html=render();for(const text of ['manual','scheduled','toString','provider error',attempt.started_at,attempt.finished_at])assert.ok(html.includes(text));assert.doesNotMatch(html,/Could not load/);
 effect();requests[2].resolve([]);await settle();assert.match(render(),/No recorded/);
 effect();requests[3].reject(new Error('timeout'));await settle();assert.match(render(),/Could not load/);
});
