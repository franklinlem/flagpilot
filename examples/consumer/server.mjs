import { createServer } from 'node:http';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { FlagPilotClient } from '@flagpilot/sdk-js';

const assets = new Map([
  ['/', ['text/html; charset=utf-8', readFileSync(new URL('./public/index.html', import.meta.url))]],
  ['/app.js', ['text/javascript; charset=utf-8', readFileSync(new URL('./public/app.js', import.meta.url))]],
  ['/style.css', ['text/css; charset=utf-8', readFileSync(new URL('./public/style.css', import.meta.url))]],
]);

// Demonstração pública LOCAL de cotação, com identidade sintética fixa no servidor.
// Uma flag altera o cálculo; nunca substitui autorização em aplicações reais.
export function createConsumer({ client, flagKey = 'shipping_v2', bucketingKey = 'demo-customer-1' }) {
  const server = createServer(async (req, res) => {
    const origin = `http://127.0.0.1:${server.address().port}`;
    const send = (status, value, type = 'application/json; charset=utf-8') => {
      res.writeHead(status, { 'Content-Type': type, 'Cache-Control': 'no-store',
        'X-Content-Type-Options': 'nosniff', 'Referrer-Policy': 'no-referrer',
        'Content-Security-Policy': "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'" });
      res.end(type.startsWith('application/json') ? JSON.stringify(value) : value);
    };
    if (req.headers.host !== new URL(origin).host) return send(403, { error: 'Host não permitido.' });
    if (req.method === 'GET' && assets.has(req.url)) {
      const [type, body] = assets.get(req.url);
      return send(200, body, type);
    }
    if (req.method !== 'POST' || req.url !== '/api/quote') return send(404, { error: 'Recurso não encontrado.' });
    if (req.headers.origin !== origin) return send(403, { error: 'Origem não permitida.' });
    if (req.headers['content-type'] !== 'application/json') return send(415, { error: 'Envie JSON.' });
    try {
      let body = '';
      for await (const chunk of req) {
        body += chunk;
        if (Buffer.byteLength(body) > 1024) { send(413, { error: 'Corpo muito grande.' }); return; }
      }
      const input = JSON.parse(body);
      if (!input || Array.isArray(input) || Object.keys(input).length !== 1 || !Number.isInteger(input.quantity) || input.quantity < 1 || input.quantity > 10) {
        return send(400, { error: 'Informe quantidade inteira entre 1 e 10.' });
      }
      const result = await client.evaluate(flagKey, { bucketingKey, defaultValue: false });
      const shippingCents = result.value ? (input.quantity >= 3 ? 0 : 800) : 1200;
      return send(200, { quantity: input.quantity, subtotalCents: input.quantity * 2000, shippingCents,
        totalCents: input.quantity * 2000 + shippingCents, experience: result.value ? 'new' : 'standard',
        fallback: result.reason.startsWith('SDK_') || ['NOT_FOUND', 'MISSING_CONTEXT'].includes(result.reason) });
    } catch {
      // Sem detalhes internos, token, contexto ou resposta bruta do SDK no browser/log.
      if (!res.headersSent) send(400, { error: 'Não foi possível processar a cotação.' });
    }
  });
  server.requestTimeout = 5000;
  server.headersTimeout = 5000;
  server.maxConnections = 32;
  return server;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  let client;
  try {
    const port = Number(process.env.DEMO_PORT ?? 3100);
    if (!Number.isInteger(port) || port < 0 || port > 65535) throw new Error('port');
    const key = process.env.FLAGPILOT_FLAG_KEY ?? 'shipping_v2';
    const context = process.env.FLAGPILOT_BUCKETING_KEY ?? 'demo-customer-1';
    if (!/^[a-z][a-z0-9_.-]{1,79}$/.test(key) || !context || Buffer.byteLength(context) > 256) throw new Error('config');
    client = new FlagPilotClient({ baseUrl: process.env.FLAGPILOT_BASE_URL ?? 'http://127.0.0.1:8080/api/v1',
      token: process.env.FLAGPILOT_EVALUATION_TOKEN, cacheTtlMs: 0, timeoutMs: 1000, maxRetries: 0 });
    const server = createConsumer({ client, flagKey: key, bucketingKey: context });
    server.on('error', () => { client.close(); process.stderr.write('Falha ao iniciar consumidor local.\n'); process.exitCode = 1; });
    server.listen(port, '127.0.0.1', () => process.stdout.write(`Consumidor: http://127.0.0.1:${server.address().port}\n`));
    const stop = () => { server.closeAllConnections(); server.close(() => client.close()); };
    process.once('SIGINT', stop);
    process.once('SIGTERM', stop);
  } catch {
    client?.close();
    process.stderr.write('Configuração inválida: confira o ambiente privado do consumidor.\n');
    process.exitCode = 1;
  }
}
