#!/usr/bin/env python3
"""API/PostgreSQL reais descartáveis, BFF Node e browser; nenhum token nos artefatos."""
import argparse
import selectors
import datetime as dt
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


def read_line(process):
    with selectors.DefaultSelector() as selector:
        selector.register(process.stdout, selectors.EVENT_READ)
        if not selector.select(timeout=45):
            raise RuntimeError('Consumer/browser excedeu tempo de resposta')
        return process.stdout.readline().strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path, help='Arquivo novo para evidência sem segredos')
    args = parser.parse_args()
    if args.report and args.report.exists():
        parser.error('Relatório já existe; escolha arquivo novo')
    project = 'flagpilot-consumer-' + secrets.token_hex(4)
    consumer = None
    with tempfile.TemporaryDirectory(prefix='flagpilot-consumer-') as tmp:
        envfile = Path(tmp) / '.env'
        envfile.write_text('POSTGRES_PASSWORD=' + secrets.token_hex(24) + '\nAPP_DB_PASSWORD=' + secrets.token_hex(24) + '\n')
        envfile.chmod(0o600)
        base = ['docker', 'compose', '--project-name', project, '--env-file', str(envfile), '-f', str(ROOT / 'deploy/compose.smoke.yml')]
        try:
            subprocess.run(base + ['up', '--build', '-d', '--wait'], cwd=ROOT, check=True)
            port = subprocess.check_output(base + ['port', 'api', '8080'], text=True).strip()
            api = 'http://' + port + '/api/v1'
            origin = 'http://127.0.0.1:5173'
            password = secrets.token_urlsafe(32)
            subprocess.run(base + ['exec', '-T', 'api', '/admin', 'bootstrap', '--email', 'consumer@example.invalid', '--org', 'SDK Consumer', '--slug', 'consumer', '--password-stdin'], input=password+'\n', text=True, stdout=subprocess.DEVNULL, check=True)
            admin = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
            headers = {'Origin': origin, 'Content-Type': 'application/json'}

            def request(path, body=None, method=None, extra=None):
                req = urllib.request.Request(api+path, data=json.dumps(body).encode() if body is not None else None, method=method, headers={**headers, **(extra or {})})
                with admin.open(req, timeout=5) as res:
                    return json.load(res), res.headers

            identity, _ = request('/auth/login', {'email': 'consumer@example.invalid', 'password': password})
            headers['X-CSRF-Token'] = identity['csrfToken']
            org = identity['memberships'][0]['organizationId']
            proj, _ = request(f'/organizations/{org}/projects', {'name': 'Consumer', 'slug': 'consumer'})
            pid = proj['id']
            envs, _ = request(f'/projects/{pid}/environments')
            eid = next(e['id'] for e in envs['items'] if e['slug'] == 'development')
            request(f'/projects/{pid}/flags', {'key': 'shipping_v2', 'description': 'Synthetic consumer shipping'})
            issued, _ = request(f'/environments/{eid}/tokens', {'name': 'Consumer ephemeral', 'expiresAt': (dt.datetime.now(dt.timezone.utc)+dt.timedelta(days=1)).isoformat()})
            secret = issued['secret']
            subprocess.run(['npm', 'run', 'build', '--workspace', '@flagpilot/sdk-js'], cwd=ROOT, check=True)
            env = {**os.environ, 'FLAGPILOT_BASE_URL': api, 'FLAGPILOT_EVALUATION_TOKEN': secret, 'DEMO_PORT': '0',
                   'FLAGPILOT_FLAG_KEY': 'shipping_v2', 'FLAGPILOT_BUCKETING_KEY': 'demo-customer-1'}
            consumer = subprocess.Popen(['node', 'examples/consumer/server.mjs'], cwd=ROOT, env=env, stdout=subprocess.PIPE, text=True)
            url = read_line(consumer).removeprefix('Consumidor: ')
            assert url.startswith('http://127.0.0.1:'), 'Consumer unavailable'
            os.environ.setdefault('PLAYWRIGHT_BROWSERS_PATH', str(ROOT / '.tools/playwright'))
            # Playwright no processo separado: recebe somente URL, nunca token de avaliação.
            browser = subprocess.Popen(['node', 'examples/consumer/browser-check.mjs'], cwd=ROOT,
                                       env={k: v for k, v in os.environ.items() if 'TOKEN' not in k and 'PASSWORD' not in k},
                                       stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            assert read_line(browser) == 'READY'
            checks = []
            for enabled, bps in [(False, 10000), (True, 10000), (True, 0), (True, 3725)]:
                path = f'/environments/{eid}/flags/shipping_v2'
                _, h = request(path)
                request(path, {'enabled': enabled, 'rolloutBps': bps}, 'PUT', {'If-Match': h['ETag']})
                req = urllib.request.Request(api+'/evaluate', data=json.dumps({'flagKey': 'shipping_v2', 'context': {'bucketingKey': 'demo-customer-1'}}).encode(), headers={'Authorization': 'Bearer '+secret, 'Content-Type': 'application/json'})
                with urllib.request.urlopen(req, timeout=5) as res:
                    expected = json.load(res)['value']
                browser.stdin.write(json.dumps({'url': url, 'enabled': expected, 'fallback': False})+'\n')
                browser.stdin.flush()
                assert read_line(browser) == 'PASS', 'Browser scenario failed'
                checks.append({'enabled': enabled, 'rolloutBps': bps, 'newExperience': expected})
            # API realmente parada: o BFF deve conservar cotação padrão pelo fallback SDK.
            subprocess.run(base + ['stop', 'api'], check=True, stdout=subprocess.DEVNULL)
            browser.stdin.write(json.dumps({'url': url, 'enabled': False, 'fallback': True})+'\n')
            browser.stdin.flush()
            assert read_line(browser) == 'PASS', 'Fallback failed'
            browser.stdin.close()
            assert browser.wait(timeout=15) == 0
            evidence = {'date': dt.datetime.now(dt.timezone.utc).date().isoformat(), 'realDisposableAPI': True, 'browser': 'chromium', 'scenarios': checks,
                        'apiStoppedFallback': True, 'browserSecretsAbsent': True, 'productionChanged': False}
            if args.report:
                with args.report.open('x') as target:
                    target.write(json.dumps(evidence, indent=2)+'\n')
            print('Consumer E2E aprovado: alternancia, rollout, BFF, browser sem segredo e fallback com API parada.')
        finally:
            if 'browser' in locals() and browser.poll() is None:
                browser.terminate()
                browser.wait(timeout=10)
            if consumer:
                consumer.terminate()
                consumer.wait(timeout=10)
            subprocess.run(base + ['down', '--volumes', '--remove-orphans'], check=True)


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('Consumer interrompido; recursos descartaveis proprios encerrados.')
        raise SystemExit(130)
