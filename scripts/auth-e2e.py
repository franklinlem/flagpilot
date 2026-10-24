#!/usr/bin/env python3
"""Browser real, dados e credenciais efêmeros; nunca usa o volume de desenvolvimento."""
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[1]

def main():
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
            envfile.write_text(config + 'PUBLIC_BASE_URL=' + origin + '\n')
            subprocess.run(base + ['up', '-d', '--wait', 'api', 'web'], check=True, cwd=ROOT)
            password = secrets.token_urlsafe(32)
            subprocess.run(base + ['exec', '-T', 'api', '/admin', 'bootstrap', '--email', 'owner@example.invalid', '--org', 'E2E Organization', '--slug', 'e2e', '--password-stdin'], input=password + '\n', text=True, check=True)
            for _ in range(30):
                try:
                    with urllib.request.urlopen(origin, timeout=2) as response:
                        if response.status == 200: break
                except OSError: time.sleep(.2)
            else: raise RuntimeError('Frontend E2E indisponível')
            env = dict(os.environ, E2E_BASE_URL=origin, E2E_EMAIL='owner@example.invalid', E2E_PASSWORD=password)
            env.setdefault('PLAYWRIGHT_BROWSERS_PATH', str(ROOT / '.tools/playwright'))
            result = subprocess.run(['npm', 'run', 'test:e2e'], cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            print(result.stdout.replace(password, '[REDACTED]'))
            result.check_returncode()
            print('E2E de autenticação aprovado em stack isolada.')
        finally:
            subprocess.run(base + ['down', '--volumes', '--remove-orphans'], check=True)

if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('E2E interrompido; recursos descartáveis próprios encerrados.')
        raise SystemExit(130)
