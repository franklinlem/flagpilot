#!/usr/bin/env python3
"""Browser real, dados e credenciais efêmeros; nunca usa o volume de desenvolvimento."""
import os
import argparse
import re
import json
import http.cookiejar
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[1]

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--browser', choices=['chromium', 'firefox'], default='chromium')
    parser.add_argument('--suite', choices=['administrative', 'security'], default='administrative')
    args = parser.parse_args()
    browser = args.browser
    project = 'flagpilot-auth-e2e-' + secrets.token_hex(4)
    with tempfile.TemporaryDirectory(prefix='flagpilot-auth-e2e-') as temp:
        envfile = Path(temp) / '.env'
        config = 'POSTGRES_PASSWORD=' + secrets.token_hex(24) + '\nAPP_DB_PASSWORD=' + secrets.token_hex(24) + '\n'
        envfile.write_text(config)
        envfile.chmod(0o600)
        base = ['docker', 'compose', '--project-name', project, '--env-file', str(envfile), '-f', str(ROOT / 'deploy/compose.smoke.yml'), '--profile', 'browser']
        try:
            subprocess.run(base + ['up', '--build', '-d', '--wait'], check=True, cwd=ROOT)
            port = subprocess.check_output(base + ['port', 'web', '8080'], text=True).strip()
            origin = 'http://' + port
            # Origem conhecida somente depois da atribuição da porta efêmera.
            # Preservar a porta ao recriar web por mudança da dependência API.
            envfile.write_text(config + 'PUBLIC_BASE_URL=' + origin + '\nBROWSER_PORT=' + port.rsplit(':',1)[1] + '\n')
            subprocess.run(base + ['up', '-d', '--wait', 'api', 'web'], check=True, cwd=ROOT)
            password = secrets.token_urlsafe(32)
            subprocess.run(base + ['exec', '-T', 'api', '/admin', 'bootstrap', '--email', 'owner@example.invalid', '--org', 'E2E Organization', '--slug', 'e2e', '--password-stdin'], input=password + '\n', text=True, check=True)
            jar = http.cookiejar.CookieJar()
            client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
            login = urllib.request.Request(origin + '/api/v1/auth/login', data=json.dumps({'email':'owner@example.invalid','password':password}).encode(), headers={'Origin':origin,'Content-Type':'application/json'})
            with client.open(login, timeout=5) as response:
                identity = json.load(response)
            organization_id = identity['memberships'][0]['organizationId']
            viewer_password = secrets.token_urlsafe(32)
            provision = subprocess.run(base + ['exec','-T','api','/admin','create-user','--email','viewer@example.invalid','--org-id',organization_id,'--password-stdin'], input=viewer_password + '\n', text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=True)
            viewer_id = next(line.split(':',1)[1].strip() for line in provision.stdout.splitlines() if line.startswith('ID do usuário:'))
            for _ in range(30):
                try:
                    with urllib.request.urlopen(origin, timeout=2) as response:
                        if response.status == 200: break
                except OSError: time.sleep(.2)
            else: raise RuntimeError('Frontend E2E indisponível')
            env = dict(os.environ, E2E_BROWSER=browser, E2E_SUITE=args.suite, E2E_BASE_URL=origin, E2E_EMAIL='owner@example.invalid', E2E_PASSWORD=password, E2E_VIEWER_EMAIL='viewer@example.invalid', E2E_VIEWER_PASSWORD=viewer_password, E2E_VIEWER_ID=viewer_id)
            env.setdefault('PLAYWRIGHT_BROWSERS_PATH', str(ROOT / '.tools/playwright'))
            subprocess.run(['npm', 'run', 'build', '--workspace', '@flagpilot/sdk-js'], cwd=ROOT, check=True)
            result = subprocess.run(['npm', 'run', 'test:e2e'], cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            output = result.stdout.replace(password, '[REDACTED]').replace(viewer_password, '[REDACTED]')
            print(re.sub(r'fp_eval_[a-f0-9]{64}', '[REDACTED_EVALUATION_TOKEN]', output))
            for artifact in (ROOT / 'apps/web/test-results').rglob('*'):
                if artifact.is_file() and artifact.suffix in {'.md', '.txt', '.json', '.html', '.log'}:
                    content = artifact.read_text(errors='replace').replace(password, '[REDACTED]').replace(viewer_password, '[REDACTED]')
                    artifact.write_text(re.sub(r'fp_eval_[a-f0-9]{64}', '[REDACTED_EVALUATION_TOKEN]', content))
            result.check_returncode()
            print(f'E2E {args.suite} aprovado em stack isolada ({browser}).')
        finally:
            subprocess.run(base + ['down', '--volumes', '--remove-orphans'], check=True)

if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('E2E interrompido; recursos descartáveis próprios encerrados.')
        raise SystemExit(130)
