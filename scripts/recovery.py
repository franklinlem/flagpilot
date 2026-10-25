#!/usr/bin/env python3
"""Ensaio de pg_dump/restore SOMENTE sintético, sem aceitar destino ou dump externo."""
import argparse
from datetime import datetime, timedelta, timezone
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parents[1]
TABLES = ['schema_migrations', 'users', 'organizations', 'memberships', 'projects', 'environments', 'flags', 'flag_configs', 'api_tokens', 'audit_events', 'sessions']
ORIGIN = 'http://127.0.0.1:5173'

def check(value, message):
    if not value:
        raise RuntimeError(message)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path, help='JSON sem dados/segredos; arquivo novo, sem sobrescrever')
    parser.add_argument('--upgrade-from-16-13', action='store_true', help='Ensaiar upgrade sintético do volume 16.13 para a imagem fixada atual antes do backup')
    args = parser.parse_args()
    if args.report and args.report.exists():
        parser.error('O relatório já existe; escolha um arquivo novo.')
    docker_env = dict(os.environ)
    for key in ['POSTGRES_PASSWORD','APP_DB_PASSWORD','RESTORE_POSTGRES_PASSWORD','RESTORE_APP_DB_PASSWORD','PUBLIC_BASE_URL','COMPOSE_FILE','COMPOSE_PROFILES','COMPOSE_PROJECT_NAME']:
        docker_env.pop(key,None)
    if docker_env.get('DOCKER_HOST') and not docker_env['DOCKER_HOST'].startswith('unix://'):
        parser.error('Ensaio exige Docker local por socket Unix; host remoto recusado.')
    context = subprocess.run(['docker','context','inspect'],env=docker_env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=15)
    check(context.returncode==0,'Não foi possível verificar o contexto Docker local.')
    check(json.loads(context.stdout)[0]['Endpoints']['docker']['Host'].startswith('unix://'),'Contexto Docker remoto recusado; ensaio somente local.')
    project = 'flagpilot-recovery-' + secrets.token_hex(4)
    previous_umask = os.umask(0o077)
    try:
        with tempfile.TemporaryDirectory(prefix='flagpilot-recovery-') as directory:
            temp = Path(directory)
            envfile = temp / '.env'
            envfile.write_text(''.join(f'{key}={secrets.token_hex(24)}\n' for key in ['POSTGRES_PASSWORD','APP_DB_PASSWORD','RESTORE_POSTGRES_PASSWORD','RESTORE_APP_DB_PASSWORD']))
            envfile.chmod(0o600)
            base = ['docker','compose','--project-name',project,'--env-file',str(envfile),'-f',str(ROOT/'deploy/compose.smoke.yml'),'-f',str(ROOT/'deploy/compose.recovery.yml'),'--profile','recovery']
            db_image = json.loads((ROOT/'deploy/images.lock.json').read_text())['db']
            upgrade_overlay = temp/'upgrade.json'
            if args.upgrade_from_16_13:
                upgrade_overlay.write_text(json.dumps({'services':{'db':{'image':'postgres:16.13-alpine@sha256:4e6e670bb069649261c9c18031f0aded7bb249a5b6664ddec29c013a89310d50'}}}))
                base.extend(['-f',str(upgrade_overlay)])
            def command(parts, *, data=None, output=None, fail=True):
                result = subprocess.run(base+parts, cwd=ROOT, env=docker_env, input=data, stdout=output or subprocess.PIPE, stderr=subprocess.PIPE, timeout=600)
                if fail and result.returncode:
                    # Não imprimir comandos com configuração, corpo HTTP, stderr SQL ou valores do dump.
                    raise RuntimeError(f'Comando Compose {parts[0]} falhou (status {result.returncode}).')
                return result
            def sql(service, statement):
                result = command(['exec','-T',service,'psql','-X','-U','flagpilot_migrator','-d','flagpilot','-v','ON_ERROR_STOP=1','-At'],data=statement.encode())
                return result.stdout.decode().strip()
            def endpoint(service):
                address = command(['port',service,'8080']).stdout.decode().strip()
                check(address.startswith('127.0.0.1:'),'Porta do ensaio deve ser loopback.')
                return 'http://'+address
            def client():
                return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
            def api(opener, url, path, *, data=None, method='GET', csrf=None, bearer=None, cookie=None, expected=200):
                headers = {'Content-Type':'application/json'}
                if csrf: headers.update({'Origin':ORIGIN,'X-CSRF-Token':csrf})
                if path == '/auth/login': headers['Origin']=ORIGIN
                if bearer: headers['Authorization']='Bearer '+bearer
                if cookie: headers['Cookie']='flagpilot_session='+cookie
                request = urllib.request.Request(url+'/api/v1'+path, data=None if data is None else json.dumps(data).encode(), method=method, headers=headers)
                try: response=opener.open(request,timeout=10)
                except urllib.error.HTTPError as error:
                    check(error.code==expected,f'HTTP inesperado no ensaio ({error.code}, esperado {expected}).')
                    return None, error.headers
                with response:
                    check(response.status==expected,'Status HTTP inesperado no ensaio.')
                    return (None if response.status==204 else json.load(response)), response.headers
            def manifest(service):
                result={}
                for table in TABLES:
                    # Nomes fixos; nenhuma entrada CLI vira SQL ou destino de restore.
                    rows=json.loads(sql(service,f"SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text COLLATE \"C\"),'[]'::jsonb) FROM {table} t;"))
                    result[table]={'rows':len(rows),'digest':hashlib.sha256(json.dumps(rows,sort_keys=True,separators=(',',':')).encode()).hexdigest()}
                return result
            try:
                print(f'Criando origem sintética isolada ({project})...',flush=True)
                if args.upgrade_from_16_13:
                    command(['pull','db'])
                    command(['build','api','migrate'])
                    command(['up','-d','--no-build','--wait','api'])
                else:
                    command(['up','--build','-d','--wait','api'])
                url=endpoint('api')
                password=secrets.token_urlsafe(32)
                command(['exec','-T','api','/admin','bootstrap','--email','recovery@example.invalid','--org','Recovery fixture','--slug','recovery-fixture','--password-stdin'],data=(password+'\n').encode())
                source=client()
                identity,headers=api(source,url,'/auth/login',method='POST',data={'email':'recovery@example.invalid','password':password})
                old_cookie=next(part.split('=',1)[1] for part in headers.get('Set-Cookie').split(';') if part.startswith('flagpilot_session='))
                csrf=identity['csrfToken'];org=identity['memberships'][0]['organizationId']
                project_data,_=api(source,url,f'/organizations/{org}/projects',method='POST',csrf=csrf,data={'name':'Recovery project','slug':'recovery-project'},expected=201)
                project_id=project_data['id']
                environments,_=api(source,url,f'/projects/{project_id}/environments')
                env=next(item['id'] for item in environments['items'] if item['slug']=='development')
                for key in ['recovery_flag','archived_flag']:
                    api(source,url,f'/projects/{project_id}/flags',method='POST',csrf=csrf,data={'key':key,'description':'Synthetic recovery fixture'},expected=201)
                _,version=api(source,url,f'/environments/{env}/flags/recovery_flag')
                def versioned(path,method,body,etag):
                    request=urllib.request.Request(url+'/api/v1'+path,data=json.dumps(body).encode(),method=method,headers={'Origin':ORIGIN,'X-CSRF-Token':csrf,'Content-Type':'application/json','If-Match':etag})
                    with source.open(request,timeout=10) as response: check(response.status==200,'Escrita versionada falhou.')
                versioned(f'/environments/{env}/flags/recovery_flag','PUT',{'enabled':True,'rolloutBps':3725},version['ETag'])
                _,version=api(source,url,f'/projects/{project_id}/flags/archived_flag')
                versioned(f'/projects/{project_id}/flags/archived_flag','PATCH',{'archived':True,'confirmProduction':True},version['ETag'])
                issued=[]
                for name in ['Active fixture','Revoked fixture']:
                    token,_=api(source,url,f'/environments/{env}/tokens',method='POST',csrf=csrf,data={'name':name,'expiresAt':(datetime.now(timezone.utc)+timedelta(days=7)).isoformat().replace('+00:00','Z')},expected=201)
                    issued.append(token)
                api(source,url,f'/environments/{env}/tokens/{issued[1]["token"]["id"]}',method='DELETE',csrf=csrf,expected=204)
                evaluation={'flagKey':'recovery_flag','context':{'bucketingKey':'synthetic-recovery-context'}}
                before,_=api(client(),url,'/evaluate',method='POST',bearer=issued[0]['secret'],data=evaluation)
                snapshot=manifest('db')
                check(all(item['rows']>0 for item in snapshot.values()),'Fixture não cobriu todas as tabelas.')
                upgrade_evidence = None
                if args.upgrade_from_16_13:
                    print('Ensaiando upgrade 16.13 para imagem atual no MESMO volume sintético...',flush=True)
                    previous_version=sql('db','SHOW server_version_num;')
                    check(previous_version=='160013','Origem do upgrade deve ser 16.13.')
                    command(['stop','api'])
                    # Remove override legado; DB agora vem do Dockerfile local corrigido.
                    upgrade_overlay.write_text('{}')
                    command(['build','db'])
                    command(['up','-d','--wait','--no-deps','--force-recreate','db'])
                    new_version=sql('db','SHOW server_version_num;')
                    check(new_version=='160015','Destino do upgrade deve ser 16.15.')
                    check(manifest('db')==snapshot,'Upgrade alterou dados do volume sintético.')
                    check(sql('db',"SELECT count(*) FROM pg_extension WHERE extname IN ('pgcrypto','btree_gist','ltree');")=='0','Extensões com requisitos especiais exigem plano próprio.')
                    check(sql('db','SELECT count(*) FROM pg_replication_slots;')=='0','Replicação exige plano próprio.')
                    command(['up','-d','--wait','--no-deps','api'])
                    url=endpoint('api')
                    after_upgrade,_=api(client(),url,'/evaluate',method='POST',bearer=issued[0]['secret'],data=evaluation)
                    check(after_upgrade==before,'Upgrade alterou avaliação/revisão.')
                    upgrade_evidence={'fromServerVersionNum':previous_version,'toServerVersionNum':new_version,'sameVolumeAllTablesPreserved':True,'evaluationStable':True,'specialExtensionsOrLogicalSlots':False}
                dump=temp/'synthetic.dump'
                started=time.monotonic()
                with dump.open('xb') as output:
                    command(['exec','-T','db','pg_dump','-U','flagpilot_migrator','-d','flagpilot','--format=custom'],output=output)
                dump_seconds=time.monotonic()-started
                check(dump.stat().st_mode & 0o777 == 0o600,'Dump deve ter permissão 0600.')
                # Revogação depois do snapshot seria desfeita pelo restore: exercitar a política.
                api(source,url,f'/environments/{env}/tokens/{issued[0]["token"]["id"]}',method='DELETE',csrf=csrf,expected=204)
                api(client(),url,'/evaluate',method='POST',bearer=issued[0]['secret'],data=evaluation,expected=401)
                print('Backup gerado; removendo origem e seu volume sintético...',flush=True)
                recovery_started=time.monotonic()
                command(['down','--volumes','--remove-orphans'])
                remaining = subprocess.run(['docker','volume','ls','--filter',f'label=com.docker.compose.project={project}','--format','{{.Name}}'],env=docker_env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=30)
                check(remaining.returncode==0 and not remaining.stdout.strip(),'Origem ainda possui volume; restore interrompido.')
                command(['up','-d','--wait','restore-db'])
                started=time.monotonic()
                with dump.open('rb') as archive:
                    result=subprocess.run(base+['exec','-T','restore-db','pg_restore','-U','flagpilot_migrator','-d','flagpilot','--no-owner','--exit-on-error','--single-transaction'],cwd=ROOT,env=docker_env,stdin=archive,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=180)
                    check(result.returncode==0,'pg_restore falhou no destino descartável.')
                restore_seconds=time.monotonic()-started
                check(manifest('restore-db')==snapshot,'Conteúdo restaurado divergiu do snapshot.')
                print('Todas as tabelas preservadas; invalidando credenciais antes de iniciar API restaurada...',flush=True)
                event_id=str(uuid.uuid4())
                sql('restore-db',f"BEGIN; UPDATE sessions SET revoked_at=coalesce(revoked_at,clock_timestamp()); UPDATE api_tokens SET revoked_at=coalesce(revoked_at,clock_timestamp()); INSERT INTO audit_events(id,organization_id,actor_label,action,resource_type,resource_id) VALUES('{event_id}','{org}','local-operator','recovery.credentials_invalidated','organization','{org}'); COMMIT;")
                command(['up','--build','-d','--wait','restore-api'])
                restored_url=endpoint('restore-api')
                api(client(),restored_url,'/health/ready')
                api(client(),restored_url,'/me',cookie=old_cookie,expected=401)
                for token in issued:
                    api(client(),restored_url,'/evaluate',method='POST',bearer=token['secret'],data=evaluation,expected=401)
                restored=client()
                identity,_=api(restored,restored_url,'/auth/login',method='POST',data={'email':'recovery@example.invalid','password':password})
                fresh,_=api(restored,restored_url,f'/environments/{env}/tokens',method='POST',csrf=identity['csrfToken'],data={'name':'Reissued after recovery','expiresAt':(datetime.now(timezone.utc)+timedelta(days=1)).isoformat().replace('+00:00','Z')},expected=201)
                after,_=api(client(),restored_url,'/evaluate',method='POST',bearer=fresh['secret'],data=evaluation)
                check(before==after,'Salt/configuração/revisão ou resultado mudou após restaurar.')
                audit_page,_=api(restored,restored_url,f'/organizations/{org}/audit-events?action=recovery.credentials_invalidated')
                check(len(audit_page['items'])==1,'Invalidação de recuperação não auditada.')
                denied=command(['exec','-T','restore-db','sh','-ec','export PGPASSWORD="$APP_DB_PASSWORD"; exec psql -X -h 127.0.0.1 -U flagpilot_app -d flagpilot -v ON_ERROR_STOP=1 -v VERBOSITY=sqlstate -c "CREATE TABLE forbidden_recovery(id integer)"'],fail=False)
                check(denied.returncode!=0 and b'42501' in denied.stderr,'Runtime recebeu DDL após restauração.')
                report={'timestampUtc':datetime.now(timezone.utc).isoformat(),'syntheticOnly':True,'postgresImage':db_image,'minorUpgrade':upgrade_evidence,'schemaVersion':5,'backupBytes':dump.stat().st_size,'dumpSeconds':round(dump_seconds,3),'restoreSqlSeconds':round(restore_seconds,3),'functionalRecoveryDrillSeconds':round(time.monotonic()-recovery_started,3),'tableRows':{name:item['rows'] for name,item in snapshot.items()},'allTableContentsPreserved':True,'sourceVolumeDestroyedBeforeRestore':True,'sessionsAndTokensInvalidatedBeforeStartup':True,'postSnapshotRevocationProtected':True,'evaluationStableAfterReissue':True,'runtimeDDLForbidden':True,'recurringBackupConfigured':False,'productionRpoRtoValidated':False}
                if args.report:
                    args.report.parent.mkdir(parents=True,exist_ok=True)
                    with args.report.open('x') as output: json.dump(report,output,indent=2);output.write('\n')
                print(json.dumps(report,indent=2))
            finally:
                command(['down','--volumes','--remove-orphans'])
    finally: os.umask(previous_umask)

if __name__=='__main__':
    try: main()
    except KeyboardInterrupt:
        print('Ensaio interrompido; recursos sintéticos próprios encerrados.')
        raise SystemExit(130)
    except RuntimeError as error:
        print(f'Ensaio não aprovado: {error} Nenhum dump é mantido.')
        raise SystemExit(1)
    except (subprocess.TimeoutExpired,urllib.error.URLError) as error:
        print(f'Ensaio não aprovado: {type(error).__name__}. Nenhum dump é mantido; confira a etapa indicada acima.')
        raise SystemExit(1)
