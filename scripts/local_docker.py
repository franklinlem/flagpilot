"""Guardas comuns dos ensaios novos; somente Docker local, saída sem segredos."""
import json
import os
import subprocess


class LocalDocker:
    def __init__(self, root):
        self.root = root
        self.env = dict(os.environ)
        for key in ['POSTGRES_PASSWORD', 'APP_DB_PASSWORD', 'PUBLIC_BASE_URL', 'COMPOSE_FILE', 'COMPOSE_PROFILES', 'COMPOSE_PROJECT_NAME']:
            self.env.pop(key, None)
        host = self.env.get('DOCKER_HOST')
        if host and not host.startswith('unix://'):
            raise RuntimeError('Docker remoto recusado; ensaio somente local.')
        context = json.loads(self.run(['context', 'inspect']).stdout)
        if not context[0]['Endpoints']['docker']['Host'].startswith('unix://'):
            raise RuntimeError('Contexto Docker remoto recusado.')

    def run(self, parts, *, data=None, fail=True, timeout=900, output=None):
        result = subprocess.run(['docker', *parts], cwd=self.root, env=self.env, input=data,
                                stdout=output or subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
        if fail and result.returncode:
            raise RuntimeError(f'Docker {parts[0]} falhou (status {result.returncode}); saída omitida para preservar segredos.')
        return result
