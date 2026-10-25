#!/usr/bin/env python3
"""Carga k6 local e sintética: não aceita URL, token ou volume externo."""
import argparse
from datetime import datetime, timedelta, timezone
import http.cookiejar
import json
import os
from pathlib import Path
import platform
import secrets
import subprocess
import tempfile
import threading
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
K6 = 'grafana/k6:1.6.1@sha256:a5ad6bc089a08d77c3ec49f3db8c6fa7a148e4073efcac44c675dbaf3568d8e1'
ORIGIN = 'http://127.0.0.1:5173'


def check(value, message):
    if not value:
        raise RuntimeError(message)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path, help='JSON agregado novo; nunca sobrescreve')
    args = parser.parse_args()
    if args.report and args.report.exists():
        parser.error('Relatório já existe; escolha outro arquivo.')
    docker_env = dict(os.environ)
    for key in ['POSTGRES_PASSWORD', 'APP_DB_PASSWORD', 'PUBLIC_BASE_URL', 'COMPOSE_FILE', 'COMPOSE_PROFILES', 'COMPOSE_PROJECT_NAME']:
        docker_env.pop(key, None)
    check(platform.system() == 'Linux', 'Este ensaio exige Docker Engine local em Linux (rede host).')
    check(not docker_env.get('DOCKER_HOST') or docker_env['DOCKER_HOST'].startswith('unix://'), 'Docker remoto recusado.')

    def docker(parts, *, data=None, fail=True, timeout=600):
        result = subprocess.run(['docker'] + parts, cwd=ROOT, env=docker_env, input=data,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
        check(not fail or result.returncode == 0, f'Docker {parts[0]} falhou (status {result.returncode}); saída omitida para preservar segredos.')
        return result

    context = json.loads(docker(['context', 'inspect']).stdout)
    check(context[0]['Endpoints']['docker']['Host'].startswith('unix://'), 'Contexto Docker remoto recusado.')
    project = 'flagpilot-load-' + secrets.token_hex(4)
    previous_umask = os.umask(0o077)
    try:
        with tempfile.TemporaryDirectory(prefix='flagpilot-load-') as directory:
            temp = Path(directory)
            envfile = temp / '.env'
            envfile.write_text(''.join(f'{key}={secrets.token_hex(24)}\n' for key in ['POSTGRES_PASSWORD', 'APP_DB_PASSWORD']))
            envfile.chmod(0o600)
            base = ['compose', '--project-name', project, '--env-file', str(envfile), '-f',
                    str(ROOT / 'deploy/compose.smoke.yml'), '-f', str(ROOT / 'deploy/compose.load.yml')]

            def compose(parts, **kwargs):
                return docker(base + parts, **kwargs)

            def api(opener, path, *, body=None, method='GET', csrf=None, etag=None, expected=200):
                headers = {'Content-Type': 'application/json'}
                if csrf:
                    headers.update({'Origin': ORIGIN, 'X-CSRF-Token': csrf})
                if path == '/auth/login':
                    headers['Origin'] = ORIGIN
                if etag:
                    headers['If-Match'] = etag
                req = urllib.request.Request(url + '/api/v1' + path, method=method, headers=headers,
                                             data=None if body is None else json.dumps(body).encode())
                with opener.open(req, timeout=10) as res:
                    check(res.status == expected, 'Status inesperado na criação da fixture.')
                    return json.load(res), res.headers

            try:
                print('Preparando stack e fixture sintética isolada...', flush=True)
                docker(['pull', K6])
                compose(['up', '--build', '-d', '--wait', 'api'])
                ports = {}
                for port in ['8080', '9090']:
                    address = compose(['port', 'api', port]).stdout.decode().strip()
                    check(address.startswith('127.0.0.1:'), 'Publicação deve ser somente loopback.')
                    ports[port] = 'http://' + address
                url = ports['8080']
                password = secrets.token_urlsafe(32)
                compose(['exec', '-T', 'api', '/admin', 'bootstrap', '--email', 'load@example.invalid',
                         '--org', 'Load fixture', '--slug', 'load-fixture', '--password-stdin'], data=(password + '\n').encode())
                opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
                identity, _ = api(opener, '/auth/login', method='POST', body={'email': 'load@example.invalid', 'password': password})
                csrf = identity['csrfToken']
                org = identity['memberships'][0]['organizationId']
                created, _ = api(opener, f'/organizations/{org}/projects', method='POST', csrf=csrf,
                                 body={'name': 'Load project', 'slug': 'load-project'}, expected=201)
                project_id = created['id']
                envs, _ = api(opener, f'/projects/{project_id}/environments')
                env = next(item['id'] for item in envs['items'] if item['slug'] == 'development')
                api(opener, f'/projects/{project_id}/flags', method='POST', csrf=csrf,
                    body={'key': 'load_flag', 'description': 'Synthetic load fixture'}, expected=201)
                _, headers = api(opener, f'/environments/{env}/flags/load_flag')
                api(opener, f'/environments/{env}/flags/load_flag', method='PUT', csrf=csrf, etag=headers['ETag'],
                    body={'enabled': True, 'rolloutBps': 10000})
                issued, _ = api(opener, f'/environments/{env}/tokens', method='POST', csrf=csrf,
                                body={'name': 'Synthetic load', 'expiresAt': (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()}, expected=201)
                secretfile = temp / 'k6.env'
                secretfile.write_text(f'LOAD_URL={url}\nLOAD_TOKEN={issued["secret"]}\nLOAD_ENV=development\n')
                secretfile.chmod(0o600)
                ids = {service: compose(['ps', '-q', service]).stdout.decode().strip() for service in ['api', 'db']}
                cpu_model = next((line.split(':', 1)[1].strip() for line in Path('/proc/cpuinfo').read_text().splitlines() if line.startswith('model name')), 'unknown')
                report = {
                    'timestampUtc': datetime.now(timezone.utc).isoformat(), 'syntheticOnly': True,
                    'sourceCommit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
                    'host': {'architecture': platform.machine(), 'kernel': platform.release(), 'cpuModel': cpu_model,
                             'logicalCpus': os.cpu_count(), 'memoryTotalKiB': int(Path('/proc/meminfo').read_text().splitlines()[0].split()[1]),
                             'loadAverageBefore': list(os.getloadavg())},
                    'dockerVersion': docker(['version', '--format', '{{.Server.Version}}']).stdout.decode().strip(),
                    'k6Image': K6, 'k6Version': docker(['run', '--rm', '--network', 'none', K6, 'version']).stdout.decode().strip(),
                    'apiImageId': docker(['inspect', '--format', '{{.Image}}', ids['api']]).stdout.decode().strip(),
                    'postgresImage': 'postgres:16.13-alpine', 'schemaVersion': 5,
                    'limits': {'apiCpus': 1, 'apiMemoryMiB': 256, 'dbCpus': 1, 'dbMemoryMiB': 512, 'generatorCpus': 1,
                               'generatorMemoryMiB': 256, 'dbPoolMax': 5, 'credentialRps': 100, 'peerRps': 120, 'globalRps': 600},
                    'workload': {'endpoint': 'POST /api/v1/evaluate', 'flags': 1, 'activeTokens': 1, 'rolloutBps': 10000,
                                 'cache': False, 'durationPerScenarioSeconds': 30, 'preAllocatedVUs': 20, 'maxVUs': 20,
                                 'network': 'same Linux host, loopback TCP; no external network', 'sharedHost': True},
                    'scenarios': [], 'productionSloValidated': False,
                }
                # Warm connection pools; separate from measurements and limiter windows.
                for _ in range(10):
                    req = urllib.request.Request(url + '/api/v1/evaluate', method='POST', data=b'{"flagKey":"load_flag"}',
                                                 headers={'Authorization': 'Bearer ' + issued['secret'], 'Content-Type': 'application/json'})
                    with urllib.request.urlopen(req, timeout=5) as res:
                        check(json.load(res)['value'] is True, 'Warmup de avaliação falhou.')
                    time.sleep(0.05)
                for rate in [10, 50, 100, 150]:
                    time.sleep(1.2)
                    print(f'Medindo {rate} req/s por 30 s (limites ativos)...', flush=True)
                    stop = threading.Event()
                    samples = []
                    sampling_errors = []

                    def monitor():
                        while not stop.is_set():
                            try:
                                snapshot = {'containers': {}}
                                raw = docker(['stats', '--no-stream', '--format', '{{json .}}', *ids.values()], timeout=10)
                                entries = [json.loads(line) for line in raw.stdout.decode().splitlines()]
                                for service, cid in ids.items():
                                    entry = next(item for item in entries if cid.startswith(item['ID']))
                                    snapshot['containers'][service] = {key: entry[key] for key in ['CPUPerc', 'MemUsage', 'MemPerc']}
                                with urllib.request.urlopen(ports['9090'] + '/metrics', timeout=3) as response:
                                    metrics = response.read().decode().splitlines()
                                snapshot['poolAcquired'] = float(next(line.split()[-1] for line in metrics if line.startswith('flagpilot_db_pool_acquired_connections ')))
                                samples.append(snapshot)
                            except Exception:
                                sampling_errors.append('sampling_failed')
                            stop.wait(2)

                    thread = threading.Thread(target=monitor, daemon=True)
                    thread.start()
                    started = time.monotonic()
                    try:
                        result = docker(['run', '--rm', '--name', project + '-k6', '--network', 'host', '--cpus', '1', '--memory', '256m',
                                         '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
                                         '--env-file', str(secretfile), '-e', f'LOAD_RATE={rate}',
                                         '-v', str(ROOT / 'scripts/load/evaluation.js') + ':/evaluation.js:ro', K6,
                                         'run', '--quiet', '--no-color', '/evaluation.js'], fail=False, timeout=70)
                        elapsed = time.monotonic() - started
                    finally:
                        stop.set()
                        thread.join(timeout=15)
                    check(not thread.is_alive(), 'Coletor não encerrou.')
                    metrics = json.loads(result.stdout)
                    scenario = {'offeredRps': rate, 'offeredIterations': rate * 30, 'wallSecondsIncludingK6Startup': round(elapsed, 3),
                                'k6ExitCode': result.returncode, 'metrics': metrics, 'resourceSamples': samples,
                                'samplingErrors': len(sampling_errors)}
                    scenario['achievedRequestsPerScheduledSecond'] = metrics['http_reqs']['count'] / 30
                    scenario['checksPassed'] = (result.returncode == 0 and metrics['evaluation_errors']['count'] == 0
                                                and metrics['dropped_iterations']['count'] == 0 and metrics['evaluation_successes']['count'] > 0
                                                and (rate >= 100 or metrics['evaluation_throttled']['count'] == 0)
                                                and (rate != 150 or metrics['evaluation_throttled']['count'] > 0)
                                                and abs(metrics['http_reqs']['count'] - rate * 30) <= 1
                                                and sum(metrics[key]['count'] for key in ['evaluation_successes', 'evaluation_throttled', 'evaluation_errors']) == metrics['http_reqs']['count'])
                    report['scenarios'].append(scenario)
                    print(f"  200 válidos: {metrics['evaluation_successes']['count']}; 429: {metrics['evaluation_throttled']['count']}; erros: {metrics['evaluation_errors']['count']}; p95 sucesso: {metrics.get('evaluation_success_wall_ms', {}).get('p(95)')} ms", flush=True)
                report['checksPassed'] = all(item['checksPassed'] and item['samplingErrors'] == 0 and item['resourceSamples'] for item in report['scenarios'])
                if args.report:
                    args.report.parent.mkdir(parents=True, exist_ok=True)
                    with args.report.open('x') as output:
                        json.dump(report, output, indent=2)
                        output.write('\n')
                check(report['checksPassed'], 'Ensaio registrou falhas; consulte relatório agregado (se solicitado).')
                print('Ensaio concluído. Recursos sintéticos serão removidos; dados de desenvolvimento preservados.', flush=True)
            finally:
                docker(['rm', '-f', project + '-k6'], fail=False, timeout=30)
                compose(['down', '--volumes', '--remove-orphans'])
    finally:
        os.umask(previous_umask)


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('Carga interrompida; limpeza dos recursos próprios executada.')
        raise SystemExit(130)
    except RuntimeError as error:
        print(f'Ensaio não aprovado: {error}')
        raise SystemExit(1)
    except (subprocess.TimeoutExpired, urllib.error.URLError, ValueError, KeyError) as error:
        # Não imprimir exceções urllib/JSON ou stdout do k6: podem carregar dados de request.
        print(f'Ensaio não aprovado ({type(error).__name__}); confira a etapa acima.')
        raise SystemExit(1)
