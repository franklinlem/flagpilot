#!/usr/bin/env python3
"""Ensaio de rollback de binário/schema 5 reconstruído de fontes locais, somente sintético."""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import http.cookiejar
import io
import json
import os
from pathlib import Path
import secrets
import subprocess
import tarfile
import tempfile
import time
import urllib.error
import urllib.request

from local_docker import LocalDocker

ROOT = Path(__file__).resolve().parents[1]
PREVIOUS = '2a05bff'  # 6B: anterior à instrumentação, também exige schema 5.
ORIGIN = 'http://127.0.0.1:5173'
TABLES = ['schema_migrations', 'users', 'organizations', 'memberships', 'projects', 'environments', 'flags', 'flag_configs', 'api_tokens', 'audit_events', 'sessions']


def check(value, message):
    if not value:
        raise RuntimeError(message)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path, help='Arquivo JSON NOVO sem dados/credenciais')
    args = parser.parse_args()
    if args.report and args.report.exists():
        parser.error('Relatório já existe; escolha outro arquivo.')
    docker = LocalDocker(ROOT)
    previous = subprocess.check_output(['git', 'rev-parse', PREVIOUS + '^{commit}'], cwd=ROOT, text=True).strip()
    candidate = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    project = 'flagpilot-rollback-' + secrets.token_hex(4)
    previous_umask = os.umask(0o077)
    tags = [project + '-previous:local', project + '-candidate:local']
    try:
        with tempfile.TemporaryDirectory(prefix='flagpilot-rollback-') as directory:
            temp = Path(directory)
            old = temp / 'previous'
            old.mkdir()
            # Apenas árvore de commit local fixo e confiável; sem checkout/reset/stash do usuário.
            archive = subprocess.check_output(['git', 'archive', previous], cwd=ROOT)
            with tarfile.open(fileobj=io.BytesIO(archive)) as source:
                source.extractall(old, filter='data')
            lock = json.loads((ROOT / 'deploy/images.lock.json').read_text())
            file = old / 'apps/api/Dockerfile'
            file.write_text(file.read_text().replace('golang:1.27.3-alpine', lock['go']))
            envfile = temp / '.env'
            envfile.write_text(''.join(f'{key}={secrets.token_hex(24)}\n' for key in ['POSTGRES_PASSWORD', 'APP_DB_PASSWORD']))
            override = temp / 'image.json'
            base = ['compose', '--project-name', project, '--env-file', str(envfile), '-f', str(ROOT / 'deploy/compose.smoke.yml'), '-f', str(override)]

            def compose(parts, **kwargs):
                return docker.run(base + parts, **kwargs)

            def sql(statement):
                return compose(['exec', '-T', 'db', 'psql', '-X', '-U', 'flagpilot_migrator', '-d', 'flagpilot', '-v', 'ON_ERROR_STOP=1', '-At'], data=statement.encode()).stdout.decode().strip()

            def manifest():
                result = {}
                for table in TABLES:
                    rows = json.loads(sql(f'SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE "C"),\'[]\'::jsonb) FROM {table} t;'))
                    result[table] = {'rows': len(rows), 'digest': hashlib.sha256(json.dumps(rows, sort_keys=True, separators=(',', ':')).encode()).hexdigest()}
                return result

            url = None
            opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

            def api(path, *, body=None, method='GET', csrf=None, bearer=None, etag=None, expected=200):
                headers = {'Content-Type': 'application/json'}
                if csrf:
                    headers.update({'Origin': ORIGIN, 'X-CSRF-Token': csrf})
                if path == '/auth/login':
                    headers['Origin'] = ORIGIN
                if bearer:
                    headers['Authorization'] = 'Bearer ' + bearer
                if etag:
                    headers['If-Match'] = etag
                request = urllib.request.Request(url + '/api/v1' + path, method=method, headers=headers,
                                                data=None if body is None else json.dumps(body).encode())
                # Avaliação não recebe cookie administrativo.
                client = urllib.request.build_opener() if bearer else opener
                try:
                    response = client.open(request, timeout=10)
                except urllib.error.HTTPError as error:
                    check(error.code == expected, f'HTTP {error.code} inesperado no ensaio (esperado {expected}).')
                    return None, error.headers
                with response:
                    check(response.status == expected, 'Status inesperado no ensaio.')
                    return (None if response.status == 204 else json.load(response)), response.headers

            def switch(tag):
                nonlocal url
                override.write_text(json.dumps({'services': {'api': {'image': tag}}}))
                started = time.monotonic()
                compose(['up', '-d', '--no-build', '--wait', '--force-recreate', '--no-deps', 'api'])
                address = compose(['port', 'api', '8080']).stdout.decode().strip()
                check(address.startswith('127.0.0.1:'), 'Porta deve ser loopback.')
                url = 'http://' + address
                running = compose(['ps', '-q', 'api']).stdout.decode().strip()
                actual = docker.run(['inspect', '--format', '{{.Image}}', running]).stdout.decode().strip()
                check(actual == image_ids[tag], 'API não usa imagem esperada após troca.')
                api('/health/ready')
                return round(time.monotonic() - started, 3)

            try:
                print(f'Construindo APIs anterior/candidata para {project}...', flush=True)
                image_ids = {}
                for tag, directory in [(tags[0], old), (tags[1], ROOT)]:
                    docker.run(['build', '-f', str(directory / 'apps/api/Dockerfile'), '-t', tag, str(directory)])
                    image_ids[tag] = docker.run(['image', 'inspect', '--format', '{{.Id}}', tag]).stdout.decode().strip()
                check(image_ids[tags[0]] != image_ids[tags[1]], 'As imagens do ensaio precisam ser distintas.')
                override.write_text(json.dumps({'services': {'api': {'image': tags[0]}}}))
                compose(['build', 'migrate', 'db'])
                compose(['up', '-d', '--no-build', '--wait', 'api'])
                switch(tags[0])
                password = secrets.token_urlsafe(32)
                compose(['exec', '-T', 'api', '/admin', 'bootstrap', '--email', 'rollback@example.invalid', '--org', 'Rollback fixture', '--slug', 'rollback-fixture', '--password-stdin'], data=(password + '\n').encode())
                identity, _ = api('/auth/login', method='POST', body={'email': 'rollback@example.invalid', 'password': password})
                csrf = identity['csrfToken']
                org = identity['memberships'][0]['organizationId']
                created, _ = api(f'/organizations/{org}/projects', method='POST', csrf=csrf, body={'name': 'Rollback project', 'slug': 'rollback-project'}, expected=201)
                project_id = created['id']
                environments, _ = api(f'/projects/{project_id}/environments')
                env = next(item['id'] for item in environments['items'] if item['slug'] == 'development')
                api(f'/projects/{project_id}/flags', method='POST', csrf=csrf, body={'key': 'rollback_flag', 'description': 'Synthetic rollback fixture'}, expected=201)

                def token(name):
                    issued, _ = api(f'/environments/{env}/tokens', method='POST', csrf=csrf, body={'name': name, 'expiresAt': (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()}, expected=201)
                    return issued

                revoked = token('Issued before upgrade')
                volume_before = compose(['ps', '-q', 'db']).stdout.decode().strip()
                print('Atualizando binário; mantendo banco e volume...', flush=True)
                upgrade_seconds = switch(tags[1])
                _, version = api(f'/environments/{env}/flags/rollback_flag')
                api(f'/environments/{env}/flags/rollback_flag', method='PUT', csrf=csrf, etag=version['ETag'], body={'enabled': True, 'rolloutBps': 3725})
                api(f'/environments/{env}/tokens/{revoked["token"]["id"]}', method='DELETE', csrf=csrf, expected=204)
                active = token('Issued by candidate')
                evaluation = {'flagKey': 'rollback_flag', 'context': {'bucketingKey': 'synthetic-rollback-context'}}
                before, _ = api('/evaluate', method='POST', body=evaluation, bearer=active['secret'])
                snapshot = manifest()
                print('Revertendo para binário anterior sem restore ou migration down...', flush=True)
                rollback_seconds = switch(tags[0])
                check(manifest() == snapshot, 'Rollback alterou dados persistentes.')
                api('/me')
                after, _ = api('/evaluate', method='POST', body=evaluation, bearer=active['secret'])
                check(before == after, 'Avaliação/revisão alterada pelo rollback.')
                api('/evaluate', method='POST', body=evaluation, bearer=revoked['secret'], expected=401)
                _, version = api(f'/environments/{env}/flags/rollback_flag')
                api(f'/environments/{env}/flags/rollback_flag', method='PUT', csrf=csrf, etag=version['ETag'], body={'enabled': False, 'rolloutBps': 3725})
                disabled, _ = api('/evaluate', method='POST', body=evaluation, bearer=active['secret'])
                check(disabled['value'] is False and disabled['reason'] == 'DISABLED', 'Escrita após rollback falhou.')
                print('Reaplicando candidata e conferindo escrita feita após rollback...', flush=True)
                reupgrade_seconds = switch(tags[1])
                persisted, _ = api('/evaluate', method='POST', body=evaluation, bearer=active['secret'])
                check(persisted == disabled, 'Reupgrade perdeu escrita feita na API anterior.')
                check(compose(['ps', '-q', 'db']).stdout.decode().strip() == volume_before, 'Container DB foi substituído indevidamente.')
                check(sql('SELECT version::text || \':\' || dirty::text FROM schema_migrations;') == '5:false', 'Schema incompatível/dirty.')
                report = {'timestampUtc': datetime.now(timezone.utc).isoformat(), 'syntheticOnly': True,
                          'previousSourceCommit': previous, 'candidateSourceCommit': candidate,
                          'candidateWorkingTreeDirty': bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT)),
                          'candidateDockerfileSha256': hashlib.sha256((ROOT / 'apps/api/Dockerfile').read_bytes()).hexdigest(),
                          'previousImageId': image_ids[tags[0]], 'candidateImageId': image_ids[tags[1]],
                          'reconstructedHistoricalSource': True, 'historicalBuildBase': lock['go'],
                          'schemaVersion': 5, 'databaseContainerPreserved': True, 'allTableContentsPreservedOnRollback': True,
                          'tableRows': {key: value['rows'] for key, value in snapshot.items()},
                          'candidateIssuedTokenAcceptedAfterRollback': True, 'revokedTokenStillRejected': True,
                          'sessionPreserved': True, 'percentageEvaluationAndRevisionStable': True,
                          'writeAfterRollbackSurvivedReupgrade': True, 'migrationDownOrRestoreUsed': False,
                          'upgradeSeconds': upgrade_seconds, 'rollbackSeconds': rollback_seconds, 'reupgradeSeconds': reupgrade_seconds,
                          'productionRollbackValidated': False, 'checksPassed': True}
                if args.report:
                    args.report.parent.mkdir(parents=True, exist_ok=True)
                    with args.report.open('x') as output:
                        json.dump(report, output, indent=2)
                        output.write('\n')
                print('Ensaio de rollback aprovado; removendo somente recursos sintéticos próprios.', flush=True)
            finally:
                # Garante override válido mesmo em falha de build anterior ao primeiro up.
                if not override.exists():
                    override.write_text(json.dumps({'services': {'api': {'image': tags[0]}}}))
                compose(['down', '--volumes', '--remove-orphans'])
                for tag in tags:
                    docker.run(['image', 'rm', tag], fail=False, timeout=30)
    finally:
        os.umask(previous_umask)


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('Rollback interrompido; limpeza dos recursos próprios executada.')
        raise SystemExit(130)
    except RuntimeError as error:
        print(str(error))
        raise SystemExit(1)
    except (subprocess.TimeoutExpired, urllib.error.URLError, ValueError, KeyError) as error:
        print(f'Ensaio não aprovado ({type(error).__name__}); confira a etapa indicada acima.')
        raise SystemExit(1)
