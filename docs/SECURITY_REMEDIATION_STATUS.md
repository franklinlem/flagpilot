# Incremento 6F — correção das imagens

Data: 25/10/2026. Correções autorizadas após o gate bloqueado de 6E. **Gate HIGH/CRITICAL aprovado nos quatro runtimes**, sem suppressions, VEX ou `--ignore-unfixed`. Não é aprovação completa de release/produção.

## Correções

- NGINX unprivileged estável 1.30.6, registro oficial atual `nginx/nginx-unprivileged`, fixado por digest. Pacotes Alpine com versões explícitas: brotli-libs 1.2.1-r0, musl/musl-utils 1.2.6-r3 e zlib 1.3.2-r1; instalação verifica a assinatura dos repositórios APK. Usuário final continua 101, porta 8080 e configuração do proxy preservadas.
- PostgreSQL 16.13 → 16.15, sem mudar major, por base fixada. Nova imagem local `deploy/postgres/Dockerfile` mantém servidor/entrypoint upstream e recompila o helper gosu 1.19 com Go 1.27.3. Fonte fixada no commit 6456aaa0f3c8 via versão Go canônica e go.sum; dependências de build isoladas em deploy/postgres/gosu, incluindo moby/sys/user 0.4.1 e x/sys 0.49.0. Não troca engine, diretório de dados ou schema.
- Migrator próprio scratch/nonroot: somente biblioteca golang-migrate 4.20.1, driver PostgreSQL e source file; elimina demais drivers e toolchain antiga da imagem genérica. Comando aplica apenas up, aceita configuração por ambiente, não imprime DSN/erros com credenciais, usa lock de 15 s e parada graciosa. Nova regressão verifica erro sem dados de conexão. O CLI administrativo em `.tools/bin/migrate` também é recompilado com Go fixado e apenas postgres/file pelo instalador; sua interface tradicional permanece disponível para manutenção controlada.
- Nenhuma dependência Go/npm da aplicação foi atualizada; biblioteca de migrations e schema 5 preservados. O módulo adicional é exclusivo do build do gosu. Compose dev/smoke/recovery, integração e ensaios passam a usar os Dockerfiles corrigidos. A chave db de images.lock.json indica a base upstream; a imagem executada é o build local, identificado no relatório por image ID.

## Novo scan/SBOM

[Resumo versionado](evidence/images-hardened-2026-10-25.json); oito arquivos completos e manifest em `.tools/releases/6f-hardened`, ignorados no Git. Scanner/DB e hashes registrados. Mesma política de 6E, com inventário obrigatório e achados unfixed incluídos.

| Runtime | Componentes | HIGH | CRITICAL | Demais achados |
|---|---:|---:|---:|---|
| API | 35 | 0 | 0 | 2 UNKNOWN, mesmo aviso de módulo em admin/server |
| Web | 71 | 0 | 0 | Nenhum |
| Migrator | 5 | 0 | 0 | Nenhum |
| PostgreSQL local | 51 | 0 | 0 | Nenhum |

As 116 ocorrências HIGH e quatro CRITICAL do snapshot 6E deixaram de ser detectadas nos runtimes corrigidos. Iteração intermediária identificou que atualizar apenas as tags não corrigia gosu/Brotli; os pacotes/helpers foram efetivamente corrigidos, sem remover metadados do scanner. Também foram corrigidos os MEDIUM/UNKNOWN com versão corrigida disponível no web/gosu.

Os dois UNKNOWN restantes são GO-2026-5932, do módulo x/crypto v0.58.0, referentes exclusivamente aos pacotes OpenPGP descontinuados. A aplicação usa outras partes de x/crypto; não remove esse módulo nem substitui criptografia de senhas. `go list -deps` de server/admin/migrate não inclui OpenPGP; govulncheck em modo binary de server e admin encontrou zero vulnerabilidades no código/imports, com um aviso no módulo requerido não chamado. O aviso bruto fica no SBOM/scan, sem supressão e sem alegar inexistência do advisory. [Advisory Go oficial](https://vuln.go.dev/ID/GO-2026-5932.json).

## Validação

- Ensaio sintético 16.13 → imagem corrigida 16.15 no mesmo volume: todas as 11 tabelas e avaliação/revisão preservadas; sem extensões especiais/slots de replicação na fixture. Em seguida dump/restore em destino vazio, invalidação de credenciais, reemissão e runtime sem DDL. Ver [recuperação/upgrade](evidence/recovery-hardened-2026-10-25.json).
- Reversão/reaplicação da API com DB corrigido e schema 5: preserva todas as tabelas, sessão, revogação, token novo, avaliação percentual e escrita após rollback. Ver [rollback](evidence/rollback-hardened-2026-10-25.json). Continua sendo reconstrução de fontes locais e schema compatível, não rollback de release publicada ou major do DB.
- Integração passou com a imagem DB corrigida. A primeira tentativa via builder legado de Testcontainers recusou os mounts BuildKit; o teste agora constrói pelo Docker CLI/BuildKit, mantém limpeza da tag própria e usa a mesma receita dos demais ambientes.
- Verificações finais e atualização do desenvolvimento registradas no fechamento abaixo. Nenhum servidor externo, deploy ou publicação foi executado.

## Operação e limites

Essas correções não demonstram ausência absoluta de vulnerabilidades nem cobrem integralmente toolchains de build, CI, host ou serviços externos. Assinatura/provenance, imagens publicadas por digest de registry, licença/release, backup recorrente/coletor/alertas e homologação externa continuam pendentes. Não houve mudança de SLO/RPO/RTO.

Antes de atualizar um banco real, revisar as notas 16.14/16.15 para extensões/replicação/limpezas específicas. O ensaio valida o esquema do FlagPilot, não toda instalação PostgreSQL possível. Não usar migration down ou restauração automática para reverter mudanças de dados.

Fontes: [PostgreSQL 16.15](https://www.postgresql.org/docs/release/16.15/), [versionamento PostgreSQL](https://www.postgresql.org/support/versioning/), [advisories NGINX](https://nginx.org/en/security_advisories.html), [NGINX unprivileged](https://github.com/nginx/docker-nginx-unprivileged), [gosu](https://github.com/tianon/gosu). Uso em [release](runbooks/release.md); decisão [ADR-016](adr/016-runtime-remediation.md).

Fechamento: `make ci` completo aprovado (Go/race/vet, integração PostgreSQL com imagem local, 21 testes frontend, 16 SDK, tipos/lint/OpenAPI e builds). Seis fluxos Chromium passaram contra novos runtimes; Firefox não foi repetido, pois não houve mudança de UI (evidência anterior permanece histórica). Testes do gate/compilação Python e oito hashes de artefatos conferidos. CLI host migrate recompilado.

Desenvolvimento atualizado localmente com volume preservado: comparação em memória do conteúdo das 11 tabelas antes/depois aprovada; PostgreSQL reporta 160015 e readiness/painel/métricas HTTP 200. [Registro agregado](evidence/development-update-2026-10-25.json). Cópia prévia real do banco em `.tools/backups/development-20261025T152734Z/before-security-update.dump`, diretório 0700/arquivo 0600 e ignorado no Git. É cópia local privada sem criptografia; arquivo legível pelo pg_restore --list, mas restore desse dump real não foi executado. Não configura rotina de backup, offsite, retenção ou alerta; ensaios de restore usaram somente dados sintéticos.

Smoke final dos containers corrigidos aprovado: live/ready/métricas, SIGINT com saída zero, reinício e perda real de DB. Recursos sintéticos removidos; desenvolvimento saudável. Nenhuma alteração externa.
