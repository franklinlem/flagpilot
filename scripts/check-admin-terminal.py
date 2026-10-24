#!/usr/bin/env python3
"""Verifica senha oculta e restauração do terminal em SIGINT, sem acessar banco."""
import errno
import os
from pathlib import Path
import pty
import select
import signal
import subprocess
import tempfile
import termios
import time

ROOT = Path(__file__).resolve().parents[1]

def main():
    env = dict(os.environ, DATABASE_URL='')
    env['PATH'] = str(ROOT / '.tools/go/bin') + ':' + env['PATH']
    with tempfile.TemporaryDirectory(prefix='flagpilot-terminal-') as directory:
        binary = str(Path(directory) / 'admin')
        subprocess.run(['go', '-C', str(ROOT / 'apps/api'), 'build', '-o', binary, './cmd/admin'], check=True, env=env)
        for interrupt in [False, True]:
            master, slave = pty.openpty()
            original = termios.tcgetattr(slave)
            process = subprocess.Popen([binary, 'bootstrap', '--email', 'terminal@example.invalid', '--org', 'Terminal', '--slug', 'terminal'], stdin=slave, stdout=slave, stderr=slave, env=env)
            output = bytearray()
            def wait_for(fragment):
                deadline = time.monotonic() + 10
                while fragment not in output:
                    if time.monotonic() > deadline: raise AssertionError('Prompt de senha indisponível')
                    if select.select([master], [], [], .1)[0]:
                        try: output.extend(os.read(master, 4096))
                        except OSError as error:
                            if error.errno != errno.EIO: raise
            try:
                wait_for(b'Senha (')
                # Prompt precede a chamada que desliga echo; aguardar essa transição real.
                deadline = time.monotonic() + 2
                while termios.tcgetattr(slave)[3] & termios.ECHO:
                    if time.monotonic() > deadline: raise AssertionError('Echo permaneceu habilitado')
                    time.sleep(.01)
                if interrupt:
                    process.send_signal(signal.SIGINT)
                    assert process.wait(timeout=5) == 130
                else:
                    password = b'terminal-test-password-123'
                    os.write(master, password + b'\n')
                    wait_for(b'Confirme a senha:')
                    time.sleep(.05)
                    os.write(master, password + b'\n')
                    # Sem URL de banco: deve falhar configuração depois de ler e confirmar.
                    assert process.wait(timeout=10) == 1
                    while select.select([master], [], [], .05)[0]:
                        try: output.extend(os.read(master, 4096))
                        except OSError: break
                    assert password not in output, 'Senha apareceu no terminal'
                    assert b'DATABASE_URL' in output
                assert termios.tcgetattr(slave) == original, 'Estado do terminal não restaurado'
            finally:
                if process.poll() is None: process.kill(); process.wait()
                os.close(master)
                os.close(slave)
    print('CLI: senha oculta, confirmação e SIGINT sem traceback aprovados.')

if __name__ == '__main__':
    main()
