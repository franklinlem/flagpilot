#!/usr/bin/env python3
"""Cria apenas configuração local; nunca sobrescreve .env existente."""
import os
import secrets
from pathlib import Path
p = Path('.env')
if p.exists():
    print('.env já existe; preservado.')
else:
    content = Path('.env.example').read_text()
    owner, runtime = secrets.token_hex(24), secrets.token_hex(24)
    content = content.replace('POSTGRES_PASSWORD=\n', f'POSTGRES_PASSWORD={owner}\n')
    content = content.replace('APP_DB_PASSWORD=\n', f'APP_DB_PASSWORD={runtime}\n')
    content = content.replace('DATABASE_URL=\n', f'DATABASE_URL=postgres://flagpilot_app:{runtime}@127.0.0.1:5432/flagpilot?sslmode=disable\n')
    fd = os.open(p, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as f: f.write(content)
    print('.env local criado com senhas aleatórias; não versionado.')
