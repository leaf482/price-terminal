import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createRequire} from 'node:module';
import {runInNewContext} from 'node:vm';
import ts from 'typescript';
import React from 'react';
import {renderToStaticMarkup} from 'react-dom/server';
import {saveMetadata} from './metadata.ts';

test('metadata requests preserve empty descriptions and never send identity fields',async()=>{
 const original=globalThis.fetch;
 try{for(const kind of ['products','retailers']){
  const form=new FormData();form.set('id','replacement');form.set('url','https://other.example');form.set('name','');form.set('brand',' Brand ');form.set('model','Model');
  globalThis.fetch=async(path,options)=>{assert.equal(path,`/api/${kind}/a%2Fb`);assert.equal(options.method,'PUT');assert.deepEqual(JSON.parse(options.body),kind==='products'?{name:'',brand:' Brand ',model:'Model'}:{name:''});return new Response('{}')};await saveMetadata(kind,'a/b',form);
  for(const status of [400,404,409,500]){globalThis.fetch=async()=>new Response('{}',{status});await assert.rejects(saveMetadata(kind,'id',form))}
 }}finally{globalThis.fetch=original}
});
function nodes(e){if(!e||typeof e!=='object')return[];if(Array.isArray(e))return e.flatMap(nodes);return[e,...nodes(e.props?.children)]}
test('metadata editor success refreshes and failure preserves input without refresh',async()=>{
 for(const kind of ['products','retailers'])for(const fail of [false,true]){
  const states=[],refs=[];let index=0,ri=0,refreshes=0,saved=0;
  const exports={},require=createRequire(import.meta.url),record={id:'original',name:'Old',brand:'Brand',model:'Model'};
  const hooks={...React,useState(v){const i=index++;if(!(i in states))states[i]=v;return[states[i],v=>states[i]=v]},useRef(v){return refs[ri++]??={current:v}}};
  const source=readFileSync(new URL('../app/catalog/metadata-editor.tsx',import.meta.url),'utf8');
  const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText;
  runInNewContext(code,{exports,Error,FormData:class{constructor(form){this.form=form}},require(name){if(name==='react')return hooks;if(name==='next/navigation')return{useRouter:()=>({refresh(){refreshes++}})};if(name.endsWith('/metadata'))return{saveMetadata:async(k,id)=>{assert.equal(k,kind);assert.equal(id,'original');if(fail)throw new Error('Save failed')}};return require(name)}});
  const render=()=>{index=ri=0;return exports.default({kind,record,onSaved(){saved++}})};
  const inputs=nodes(render()).filter(n=>n.type==='input');assert.deepEqual(inputs.map(n=>n.props.name),kind==='products'?['name','brand','model']:['name']);assert.ok(inputs.every(n=>!n.props.required));
  await nodes(render()).find(n=>n.type==='form').props.onSubmit({preventDefault(){},currentTarget:{}});
  const html=renderToStaticMarkup(render());assert.match(html,/ID \(read-only\): original/);
  assert.match(html,fail?/Save failed/:/Metadata saved/);assert.equal(refreshes,fail?0:1);assert.equal(saved,fail?0:1);
 }
});
