#!/usr/bin/env python3
"""Carrega .env sem executar shell; ambiente explícito tem precedência."""
import os
from pathlib import Path
import sys
root = Path(__file__).resolve().parents[1]
p = root / '.env'
if p.exists():
    for line in p.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith('#'): continue
        key, value = line.split('=', 1)
        os.environ.setdefault(key, value)
os.environ['PATH'] = str(root / '.tools/go/bin') + ':' + str(root / '.tools/bin') + ':' + os.environ['PATH']
os.chdir(root)
args = sys.argv[1:]
if not args: raise SystemExit('Informe um comando')
os.execvp(args[0], args)
