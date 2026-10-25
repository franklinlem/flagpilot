import { TextEncoder } from 'node:util';
import { URL } from 'node:url';
import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { createHash } from 'node:crypto';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { bucketV1 } from '../dist/bucketing.js';
const cjs = createRequire(import.meta.url)('../dist/cjs/bucketing.js');
const fixture = JSON.parse(await readFile(new URL('../../../testdata/bucketing-v1.json', import.meta.url), 'utf8'));
test('bucketing v1: vetores UTF-8 compartilhados em ESM e CJS sem perda uint64', async () => {
  assert.equal(fixture.version, 1);
  for (const row of fixture.vectors) {
    const args = [row.projectId, row.environmentId, row.flagKey, row.salt, row.bucketingKey];
    assert.equal(createHash('sha256').update(new TextEncoder().encode(args.join(':'))).digest('hex'), row.digest);
    assert.equal(await bucketV1(...args), row.bucket, row.name);
    assert.equal(await cjs.bucketV1(...args), row.bucket, row.name);
  }
  assert.notEqual(fixture.vectors[2].bucket, fixture.vectors[3].bucket, 'não normalizar Unicode');
});
