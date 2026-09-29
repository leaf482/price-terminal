import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { runInNewContext } from 'node:vm';
import ts from 'typescript';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import * as prices from './prices.ts';

// Execute the real component/effect without adding a DOM test dependency.
// The hook harness preserves state while explicitly replaying setup/cleanup,
// as Strict Mode does. Assertions inspect the component's rendered markup.
function harness() {
  const states = [], requests = [];
  let cursor = 0, effect;
  const hooks = {
    ...React,
    useState(initial) {
      const index = cursor++;
      if (!(index in states)) states[index] = initial;
      return [states[index], value => { states[index] = value; }];
    },
    useEffect(setup) { effect = setup; },
  };
  const source = readFileSync(new URL('../app/products/[id]/history-view.tsx', import.meta.url), 'utf8');
  const compiled = ts.transpileModule(source + '\nexport { HistoryResult };', {
    compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, target: ts.ScriptTarget.ES2022 },
  }).outputText;
  const require = createRequire(import.meta.url);
  const exports = {};
  runInNewContext(compiled, {
    exports, AbortController,
    setTimeout: (...args) => setTimeout(...args),
    clearTimeout: timer => clearTimeout(timer),
    require(name) {
      if (name === 'react') return hooks;
      if (name === '../../../lib/prices') return prices;
      if (name === '../../../lib/api') return {
        api: (path, _parse, signal) => new Promise((resolve, reject) => {
          requests.push({ path, signal, resolve, reject });
          signal.addEventListener('abort', () => reject(new Error('aborted')), { once: true });
        }),
      };
      return require(name);
    },
  });
  return {
    requests,
    render() { cursor = 0; return renderToStaticMarkup(exports.HistoryResult({ id: 'listing-a', range: '1M' })); },
    setup() { return effect(); },
  };
}
const history = {
  listing_id: 'listing-a', truncated: false,
  observations: [{ observed_at: '2026-09-29T12:00:00Z', source: 'fixture', stock: 'unknown' }],
  historical_low: null, period_change_percent: null,
};
// Drain then/catch/finally chains without real-time sleeps.
async function settle() { for (let i = 0; i < 8; i++) await Promise.resolve(); }

test('cleanup replay cannot turn replacement success into an error', async () => {
  const h = harness();
  assert.match(h.render(), /Loading history/);
  const cleanup = h.setup();
  assert.equal(h.requests.length, 1);
  cleanup();
  assert.equal(h.requests[0].signal.aborted, true);
  const replacementCleanup = h.setup();
  try {
    await settle();
    assert.match(h.render(), /Loading history/);
    h.requests[1].resolve(history);
    await settle();
    const html = h.render();
    assert.match(html, /Historical low/);
    assert.match(html, /Stock-only or reference-only history/);
    assert.doesNotMatch(html, /role="alert"|Loading history/);
  } finally { replacementCleanup(); }
});

test('a late success from a cleaned-up effect is ignored', async () => {
  const h = harness(); h.render();
  const cleanup = h.setup();
  // Simulate completion queued just before cancellation.
  h.requests[0].resolve(history);
  cleanup();
  const replacementCleanup = h.setup();
  try { await settle(); assert.match(h.render(), /Loading history/); }
  finally { replacementCleanup(); await settle(); }
});

test('genuine request failure renders an error and later success clears it', async () => {
  const h = harness(); h.render();
  const cleanup = h.setup();
  h.requests[0].reject(new Error('network failure'));
  await settle();
  assert.match(h.render(), /role="alert"/);
  cleanup();
  const replacementCleanup = h.setup();
  try {
    h.requests[1].resolve(history); await settle();
    assert.match(h.render(), /Historical low/);
    assert.doesNotMatch(h.render(), /role="alert"/);
  } finally { replacementCleanup(); }
});

test('timeout of the active effect still renders an error', async t => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const h = harness(); h.render();
  const cleanup = h.setup();
  try {
    t.mock.timers.tick(10000); await settle();
    assert.equal(h.requests[0].signal.aborted, true);
    assert.match(h.render(), /role="alert"/);
  } finally { cleanup(); }
});
