import { chromium } from '@playwright/test';
import { createInterface } from 'node:readline';
import assert from 'node:assert/strict';

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();
let origin;
let leaked = false;
const secretPattern = /fp_eval_[a-f0-9]{64}/;
page.on('request', request => {
  if (new URL(request.url()).origin !== origin || secretPattern.test(JSON.stringify(request.headers()) + (request.postData() ?? ''))) leaked = true;
});
const responses = [];
page.on('response', response => responses.push(response.text().then(body => { if (secretPattern.test(body)) leaked = true; })));
process.stdout.write('READY\n');
try {
  for await (const line of createInterface({ input: process.stdin })) {
    const scenario = JSON.parse(line);
    origin = scenario.url;
    await page.goto(origin);
    await page.getByLabel('Quantidade de itens').fill('3');
    const resultPromise = page.waitForResponse(response => response.url() === origin+'/api/quote');
    await page.getByRole('button', { name: 'Calcular cotação' }).click();
    const result = await (await resultPromise).json();
    assert.equal(result.experience, scenario.enabled ? 'new' : 'standard');
    assert.equal(result.shippingCents, scenario.enabled ? 0 : 1200);
    assert.equal(result.fallback, scenario.fallback);
    await page.getByRole('status').filter({ hasText: scenario.enabled ? 'Experiência nova' : 'Experiência padrão' }).waitFor();
    await Promise.all(responses.splice(0));
    assert.equal(leaked, false);
    assert.equal(await page.evaluate(() => globalThis.localStorage.length+globalThis.sessionStorage.length), 0);
    process.stdout.write('PASS\n');
  }
} finally { await browser.close(); }
