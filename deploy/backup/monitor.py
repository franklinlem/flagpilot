#!/usr/bin/env python3
"""Estado local e supervisão de backups. Somente biblioteca padrão."""
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import sys
import tempfile
import time
import urllib.parse
import urllib.request

ROOT = Path.home() / 'backups/flagpilot/homolog'
NAME = re.compile(r'flagpilot-homolog-\d{8}T\d{6}Z\.dump\.gpg')
UTC = dt.timezone.utc


def atomic(root, name, value):
    fd, tmp = tempfile.mkstemp(prefix='.status-', dir=root)
    try:
        with os.fdopen(fd, 'w') as out:
            json.dump(value, out)
            out.write('\n')
        os.replace(tmp, root / name)
    finally:
        Path(tmp).unlink(missing_ok=True)


def read(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > 4096:
        raise ValueError('metadata')
    return json.loads(path.read_text())


def timestamp(value):
    parsed = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
    if parsed.utcoffset() != dt.timedelta(0):
        raise ValueError('timezone')
    return parsed


def record(root, state, name=None):
    now = dt.datetime.now(UTC).isoformat()
    value = {'state': state, 'at': now}
    if state == 'success':
        if not name or not NAME.fullmatch(name):
            raise ValueError('filename')
        value['file'] = name
        atomic(root, 'last-success.json', value)
        # Compatibilidade com o runbook anterior; não usado como prova pelo monitor.
        (root / 'last-success.txt').write_text(now + '\n')
        (root / 'last-success.txt').chmod(0o600)
    atomic(root, 'last-attempt.json', value)


def evaluate(root, now=None):
    now = now or dt.datetime.now(UTC)
    try:
        attempt = read(root / 'last-attempt.json')
        age = (now - timestamp(attempt['at'])).total_seconds()
        if age < -300:
            return False, 'Relogio inconsistente'
        if attempt['state'] == 'failed':
            return False, 'Ultima tentativa falhou'
        if attempt['state'] not in ('running', 'success'):
            return False, 'Estado de tentativa invalido'
        if attempt['state'] == 'running' and age > 1800:
            return False, 'Tentativa excedeu 30 minutos'
        success = read(root / 'last-success.json')
        age = (now - timestamp(success['at'])).total_seconds()
        if age < -300:
            return False, 'Relogio inconsistente'
        if age > 26 * 3600:
            return False, 'Backup atrasado mais de 26 horas'
        name = success['file']
        if success['state'] != 'success' or not NAME.fullmatch(name):
            return False, 'Registro de sucesso invalido'
        path = root / name
        checksum = root / (name + '.sha256')
        if path.is_symlink() or not path.is_file() or path.stat().st_size == 0:
            return False, 'Arquivo de backup ausente'
        if checksum.is_symlink() or not checksum.is_file() or checksum.stat().st_size > 4096:
            return False, 'Checksum ausente'
        expected = checksum.read_text().split()[0]
        if not re.fullmatch(r'[a-f0-9]{64}', expected):
            return False, 'Checksum invalido'
        digest = hashlib.sha256()
        deadline = time.monotonic() + 15
        with path.open('rb') as source:
            while chunk := source.read(1024 * 1024):
                digest.update(chunk)
                if time.monotonic() > deadline:
                    return False, 'Verificacao excedeu tempo limite'
        if digest.hexdigest() != expected:
            return False, 'Checksum divergente'
        return True, 'Backup recente e checksum valido'
    except (OSError, ValueError, KeyError, TypeError, IndexError, AttributeError):
        return False, 'Metadados de backup ausentes ou invalidos'


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def push(root, status):
    config = root / 'monitor.json'
    if stat.S_IMODE(config.stat().st_mode) != 0o600:
        raise ValueError('permissions')
    url = read(config)['push_url']
    parts = urllib.parse.urlsplit(url)
    if (parts.scheme != 'http' or parts.netloc != '127.0.0.1:3001'
            or not re.fullmatch(r'/api/push/[A-Za-z0-9_-]{16,128}', parts.path)
            or parts.query or parts.fragment):
        raise ValueError('endpoint')
    query = urllib.parse.urlencode({'status': 'up' if status[0] else 'down', 'msg': status[1]})
    with urllib.request.build_opener(NoRedirect).open(url + '?' + query, timeout=5) as response:
        if json.loads(response.read(4096)).get('ok') is not True:
            raise ValueError('push')


def rotate(root):
    path = root / 'pull.log'
    if path.is_symlink() or not path.is_file() or path.stat().st_size < 1024 * 1024:
        return
    # copytruncate conserva o descritor aberto pelo cron; dois históricos próprios.
    older = root / 'pull.log.1'
    if older.is_symlink():
        raise ValueError('log symlink')
    if older.is_file() and not older.is_symlink():
        os.replace(older, root / 'pull.log.2')
    with path.open('rb+') as source:
        with older.open('wb') as target:
            os.chmod(older, 0o600)
            source.seek(max(0, path.stat().st_size - 1024 * 1024))
            target.write(source.read(1024 * 1024))
        source.truncate(0)


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] not in ('running', 'failed', 'success'):
            raise ValueError('state')
        record(ROOT, sys.argv[1], sys.argv[2] if len(sys.argv) == 3 else None)
        return
    result = evaluate(ROOT)
    delivered = False
    try:
        push(ROOT, result)
        delivered = True
    except Exception:
        # Nunca registrar URL/token, resposta de provedor ou traceback secreto.
        pass
    atomic(ROOT, 'last-watch.json', {'at': dt.datetime.now(UTC).isoformat(),
                                   'healthy': result[0], 'message': result[1], 'delivered': delivered})
    rotate(ROOT)
    if not delivered:
        sys.exit(1)


if __name__ == '__main__':
    try:
        main()
    except Exception:
        print('Supervisao de backup falhou; conferir arquivos privados.', file=sys.stderr)
        sys.exit(1)
