import { test } from 'node:test';
import assert from 'node:assert/strict';
import { jsonRequest } from './json-request.ts';
import { api } from './api.ts';
import { saveAlert } from './alerts.ts';
import { setTracking } from './tracking.ts';

test('JSON mutation transport preserves methods, bodies, headers and raw response', async () => {
 const old = globalThis.fetch;
 try {
  for (const method of ['POST', 'PATCH', 'PUT']) {
   globalThis.fetch = async (url, init) => {
    assert.equal(url, '/api/listings/constructor');
    assert.equal(init.method, method);
    assert.deepEqual(init.headers, {'Content-Type':'application/json'});
    assert.equal(init.body, '{"amount":0,"enabled":false}');
    assert.ok(init.signal instanceof AbortSignal);
    return new Response('{"data":{"amount":0}}');
   };
   const response = await jsonRequest('/api/listings/constructor', method, {amount:0, enabled:false});
   assert.deepEqual(await response.json(), {data:{amount:0}});
  }
  globalThis.fetch = async (_, init) => { assert.equal(init.body, undefined); return new Response(null, {status:204}); };
  assert.equal((await jsonRequest('/api/action','POST',undefined)).status,204);
 } finally { globalThis.fetch = old; }
});

test('feature modules keep non-2xx errors and distinct successful-body policies', async () => {
 const old = globalThis.fetch;
 try {
  globalThis.fetch = async () => new Response('private detail', {status:409});
  await assert.rejects(saveAlert('l',{}), /Could not save alert \(409\)/);
  globalThis.fetch = async () => new Response('not JSON');
  await assert.rejects(saveAlert('l',{}), SyntaxError);
  globalThis.fetch = async () => new Response(null,{status:204});
  await setTracking('l',false); // This endpoint has always ignored success bodies.
  await assert.rejects(saveAlert('l',{}), SyntaxError); // Alerts require a body.
  globalThis.fetch = async () => new Response('not JSON');
  await assert.rejects(api('/products', x=>x), SyntaxError);
  globalThis.fetch = async () => new Response('{}',{status:500});
  await assert.rejects(api('/products', x=>x), /Request failed \(500\)/);
 } finally { globalThis.fetch = old; }
});

test('timeout and caller cancellation propagate without replacing signals or swallowing errors', async () => {
 const oldFetch=globalThis.fetch, oldTimeout=AbortSignal.timeout;
 const timeoutController=new AbortController(); const durations=[];
 try {
  AbortSignal.timeout = ms => { durations.push(ms); return timeoutController.signal; };
  globalThis.fetch = (_, init) => new Promise((resolve,reject)=>{
   if(init.signal.aborted) {reject(init.signal.reason);return;}
   init.signal.addEventListener('abort',()=>reject(init.signal.reason),{once:true});
  });
  const pending=jsonRequest('/api/action','POST',{} ,30000);
  const timeout=new DOMException('Timed out','TimeoutError');
  timeoutController.abort(timeout);
  await assert.rejects(pending, e=>e===timeout);
  assert.deepEqual(durations,[30000]);
  const caller=new AbortController();
  const request=jsonRequest('/api/action','PATCH',{},10000,caller.signal);
  caller.abort();
  await assert.rejects(request,e=>e===caller.signal.reason);
  await assert.rejects(jsonRequest('/api/action','POST',{},10000,caller.signal),e=>e===caller.signal.reason);
  assert.deepEqual(durations,[30000]);
 } finally {globalThis.fetch=oldFetch;AbortSignal.timeout=oldTimeout;}
});
