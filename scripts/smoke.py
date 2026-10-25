#!/usr/bin/env python3
"""Smoke do runtime em containers isolados do desenvolvimento."""
import json
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
def main():
    project = 'flagpilot-smoke-' + secrets.token_hex(4)
    with tempfile.TemporaryDirectory(prefix='flagpilot-smoke-') as temp:
        envfile = Path(temp) / '.env'
        envfile.write_text('POSTGRES_PASSWORD=' + secrets.token_hex(24) + '\nAPP_DB_PASSWORD=' + secrets.token_hex(24) + '\n')
        envfile.chmod(0o600)
        base = ['docker','compose','--project-name', project,'--env-file',str(envfile),'-f',str(ROOT/'deploy/compose.smoke.yml')]
        try:
            subprocess.run(base + ['up','--build','-d','--wait'], check=True, cwd=ROOT)
            # Probe publicada aleatoriamente somente em loopback.
            port = subprocess.check_output(base + ['port','api','8080'], text=True).strip()
            url = 'http://' + port + '/api/v1/health/'
            for endpoint in ['live', 'ready']:
                with urllib.request.urlopen(url + endpoint, timeout=5) as response:
                    assert response.status == 200 and json.load(response) == {'status':'ok'}
            metrics_port = subprocess.check_output(base + ['port','api','9090'],text=True).strip()
            with urllib.request.urlopen('http://' + metrics_port + '/metrics',timeout=5) as response:
                metrics = response.read().decode()
                assert 'flagpilot_http_requests_total' in metrics and 'flagpilot_db_pool_max_connections 5' in metrics
            try:
                urllib.request.urlopen(url.replace('/health/','/metrics'),timeout=5)
                raise AssertionError('Métricas expostas na API principal')
            except urllib.error.HTTPError as error: assert error.code == 404
            # Colisão real no listener de métricas deve encerrar a segunda API.
            collision = subprocess.run(base + ['exec','-T','-e','HTTP_ADDR=0.0.0.0:8081','-e','METRICS_ADDR=0.0.0.0:9090','api','/server'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=15)
            assert collision.returncode == 1, 'falha de bind não encerrou a API'
            # Ctrl+C/SIGINT deve encerrar API limpa; reinício deve recuperar readiness.
            subprocess.run(base + ['kill','-s','SIGINT','api'],check=True)
            api_id = subprocess.check_output(base + ['ps','-a','-q','api'],text=True).strip()
            for _ in range(30):
                state = json.loads(subprocess.check_output(['docker','inspect',api_id],text=True))[0]['State']
                if not state['Running']: break
                time.sleep(.2)
            assert not state['Running'] and state['ExitCode'] == 0, 'SIGINT não encerrou com sucesso'
            subprocess.run(base + ['up','-d','--wait','api'],check=True)
            # Docker pode atribuir outra porta efêmera após reiniciar o container.
            port = subprocess.check_output(base + ['port','api','8080'], text=True).strip()
            url = 'http://' + port + '/api/v1/health/'
            with urllib.request.urlopen(url + 'ready',timeout=5) as response: assert response.status == 200
            # Perda real do DB não pode afetar liveness; readiness deve falhar.
            subprocess.run(base + ['stop','db'],check=True)
            with urllib.request.urlopen(url + 'live',timeout=5) as response: assert response.status == 200
            try:
                urllib.request.urlopen(url + 'ready',timeout=5)
                raise AssertionError('readiness não detectou perda do banco')
            except urllib.error.HTTPError as error:
                assert error.code == 503
            metrics_port = subprocess.check_output(base + ['port','api','9090'],text=True).strip()
            with urllib.request.urlopen('http://' + metrics_port + '/metrics',timeout=5) as response: assert response.status == 200
            print('Smoke aprovado: live/ready, SIGINT, reinício e indisponibilidade do banco.')
        except subprocess.CalledProcessError:
            logs = subprocess.run(base + ['logs', '--no-color', '--tail', '50', 'db', 'migrate', 'api'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True).stdout
            for line in envfile.read_text().splitlines():
                if 'PASSWORD=' in line: logs = logs.replace(line.split('=', 1)[1], '[REDACTED]')
            print(logs)
            raise
        finally:
            subprocess.run(base + ['down','--volumes','--remove-orphans'],check=True)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("Smoke interrompido; recursos descartáveis próprios encerrados.")
        raise SystemExit(130)
