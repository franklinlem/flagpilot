#!/usr/bin/env python3
"""Ferramentas fixadas, somente em .tools; sem sudo e com verificação SHA-256."""
import hashlib
import io
import json
from pathlib import Path
import platform
import subprocess
import sys
import tarfile
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
DEST = ROOT / '.tools'
DEST.mkdir(exist_ok=True)

def fetch(url):
    with urllib.request.urlopen(url, timeout=120) as res: return res.read()

def extract_binary(data, name):
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
        member = next(m for m in archive.getmembers() if Path(m.name).name == name and m.isfile())
        target = DEST / 'bin' / name
        target.parent.mkdir(exist_ok=True)
        target.write_bytes(archive.extractfile(member).read())
        target.chmod(0o755)

def release(repo, version, filename, binary):
    if (DEST / 'bin' / binary).exists():
        print(binary, 'já existe em .tools; preservado.'); return
    meta = json.loads(fetch(f'https://api.github.com/repos/{repo}/releases/tags/{version}'))
    asset = next(a for a in meta['assets'] if a['name'] == filename)
    data = fetch(asset['browser_download_url'])
    digest = hashlib.sha256(data).hexdigest()
    if asset.get('digest'):
        expected = asset['digest'].removeprefix('sha256:')
    else:
        checksum = next(a for a in meta['assets'] if 'checksums' in a['name'].lower() or a['name'] == 'sha256sum.txt')
        expected = next(line.split()[0] for line in fetch(checksum['browser_download_url']).decode().splitlines() if line.split()[-1].lstrip('*') == filename)
    if digest != expected: raise RuntimeError('Checksum inválido para ' + binary)
    extract_binary(data, binary)
    print(binary, version, 'instalado em .tools/bin')

if platform.system() != 'Linux' or platform.machine() != 'x86_64':
    raise SystemExit('Instalador inicial suporta Debian/Linux amd64; outros sistemas: instalar versões do README manualmente.')
flags = sys.argv[1:]
if not flags or '--go' in flags:
    if not (DEST / 'go/bin/go').exists():
        filename = 'go1.27.3.linux-amd64.tar.gz'
        data = fetch('https://go.dev/dl/' + filename)
        if hashlib.sha256(data).hexdigest() != '65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40':
            raise RuntimeError('Checksum Go inválido')
        with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
            archive.extractall(DEST, filter='data')
        print('Go 1.27.3 instalado em .tools/go')
if not flags or '--sqlc' in flags:
    release('sqlc-dev/sqlc', 'v1.31.1', 'sqlc_1.31.1_linux_amd64.tar.gz', 'sqlc')
if not flags or '--migrate' in flags:
    go_binary = DEST / 'go/bin/go'
    if not go_binary.exists():
        raise RuntimeError('Instale primeiro o Go fixado com --go para compilar migrate.')
    subprocess.run([str(go_binary), 'build', '-tags', 'postgres', '-trimpath', '-ldflags=-s -w -X main.Version=v4.20.1', '-o', str(DEST/'bin/migrate'), 'github.com/golang-migrate/migrate/v4/cmd/migrate'], cwd=ROOT/'apps/api', check=True)
    print('migrate v4.20.1 recompilado com Go fixado; somente postgres/file')
if not flags or '--gitleaks' in flags:
    release('gitleaks/gitleaks', 'v8.30.1', 'gitleaks_8.30.1_linux_x64.tar.gz', 'gitleaks')
