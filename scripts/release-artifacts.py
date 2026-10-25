#!/usr/bin/env python3
"""Build, SBOM e scan locais sem publicar imagens. HIGH/CRITICAL bloqueiam o gate."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile

from local_docker import LocalDocker
from image_evidence import summarize

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=ROOT / '.tools/releases/current', help='Diretório NOVO para SBOMs/relatórios (não é registry)')
    parser.add_argument('--report', type=Path, help='Resumo agregado em arquivo NOVO')
    args = parser.parse_args()
    if args.output.exists() or (args.report and args.report.exists()):
        parser.error('Saída/relatório já existe; escolha caminho novo.')
    docker = LocalDocker(ROOT)
    lock = json.loads((ROOT / 'deploy/images.lock.json').read_text())
    args.output = args.output.resolve()
    args.output.mkdir(parents=True, mode=0o700)
    cache = ROOT / '.tools/trivy-cache'
    cache.mkdir(parents=True, exist_ok=True, mode=0o700)
    run_id = 'flagpilot-release-' + secrets.token_hex(4)
    built = []
    scanner_name = run_id + '-scan'
    report = {'timestampUtc': datetime.now(timezone.utc).isoformat(),
              'sourceCommit': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
              'workingTreeDirty': bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT)),
              'scannerImage': lock['scanner'], 'threshold': ['HIGH', 'CRITICAL'], 'ignoreUnfixed': False,
              'published': False, 'images': {}}
    # Scanner sees only one exported image, its own cache and output. No socket/repo/env.
    scratch = tempfile.TemporaryDirectory(prefix='trivy-work-', dir=ROOT / '.tools')
    scanner = ['run', '--rm', '--name', scanner_name, '--user', f'{os.getuid()}:{os.getgid()}',
               '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
               '--cpus', '1', '--memory', '1g', '-v', f'{scratch.name}:/tmp',
               '-v', f'{cache}:/cache']
    try:
        print('Baixando scanner fixado e base de vulnerabilidades...', flush=True)
        docker.run(['pull', lock['scanner']])
        docker.run(scanner + [lock['scanner'], 'image', '--cache-dir', '/cache', '--download-db-only',
                              '--db-repository', 'public.ecr.aws/aquasecurity/trivy-db:2'], timeout=1200)
        version = docker.run(scanner + ['--network', 'none', lock['scanner'], '--cache-dir', '/cache', '--version', '--format', 'json'])
        report['scannerVersionAndDatabase'] = json.loads(version.stdout)
        targets = {}
        for service in ['api', 'web', 'migrate', 'db']:
            print(f'Construindo runtime {service}...', flush=True)
            tag = f'{run_id}-{service}:local'
            built.append(tag)
            dockerfile = {'migrate':'deploy/migrate/Dockerfile','db':'deploy/postgres/Dockerfile'}.get(service, f'apps/{service}/Dockerfile')
            docker.run(['build', '--target', 'runtime', '-f', dockerfile, '-t', tag, '.'])
            targets[service] = tag
        for service, target in targets.items():
            print(f'Gerando SBOM e scan de {service}...', flush=True)
            image_id = docker.run(['image', 'inspect', '--format', '{{.Id}}', target]).stdout.decode().strip()
            with tempfile.TemporaryDirectory(prefix='flagpilot-scan-') as directory:
                archive = Path(directory) / 'image.tar'
                with archive.open('xb') as output:
                    docker.run(['save', image_id], output=output)
                archive.chmod(0o600)
                for format_name, suffix in [('json', 'scan.json'), ('cyclonedx', 'cdx.json')]:
                    output = f'{service}.{suffix}'
                    docker.run(scanner + ['--network', 'none', '-v', f'{archive}:/input/image.tar:ro',
                                          '-v', f'{args.output}:/output', lock['scanner'], 'image', '--cache-dir', '/cache',
                                          '--skip-db-update', '--skip-java-db-update', '--scanners', 'vuln', '--list-all-pkgs',
                                          '--input', '/input/image.tar', '--format', format_name, '--output', f'/output/{output}'])
            scan = json.loads((args.output / f'{service}.scan.json').read_text())
            sbom = json.loads((args.output / f'{service}.cdx.json').read_text())
            entry = summarize(scan, sbom)
            entry.update({'imageId': image_id, 'upstreamReference': lock.get(service) if service == 'db' else None,
                          'artifacts': [{'file': f'{service}.{suffix}', 'sha256': digest(args.output / f'{service}.{suffix}')} for suffix in ['scan.json', 'cdx.json']]})
            report['images'][service] = entry
            print(f"  {service}: {entry['sbomComponents']} componentes; severidades {entry['severityCounts']}; gate={entry['gatePassed']}", flush=True)
        report['gatePassed'] = all(item['gatePassed'] for item in report['images'].values())
        with (args.output / 'manifest.json').open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        if args.report:
            args.report.parent.mkdir(parents=True, exist_ok=True)
            with args.report.open('x') as output:
                json.dump(report, output, indent=2)
                output.write('\n')
        print('Artefatos gerados; gate ' + ('aprovado.' if report['gatePassed'] else 'BLOQUEADO por HIGH/CRITICAL.'), flush=True)
        return 0 if report['gatePassed'] else 2
    finally:
        docker.run(['rm', '-f', scanner_name], fail=False, timeout=30)
        for tag in built:
            docker.run(['image', 'rm', tag], fail=False, timeout=30)
        scratch.cleanup()


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print('Scan interrompido; limpeza dos recursos próprios executada.')
        raise SystemExit(130)
    except RuntimeError as error:
        print(str(error))
        raise SystemExit(1)
    except (subprocess.TimeoutExpired, ValueError, KeyError) as error:
        print(f'Falha de ferramenta/artefato ({type(error).__name__}); gate não aprovado.')
        raise SystemExit(1)
