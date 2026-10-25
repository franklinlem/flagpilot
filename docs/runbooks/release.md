# Evidências de imagens e rollback local

Incremento 6E. Preparação local, sem release pública, registry, npm, deploy, DNS ou alteração de servidor externo. Gate de segurança atual **bloqueado**: ver [RELEASE_STATUS.md](../RELEASE_STATUS.md). Um scan sem HIGH/CRITICAL também não aprova sozinho uma release.

## SBOM e scan

```bash
make release-artifacts
# Para outra execução, escolher um diretório NOVO:
make release-artifacts ARGS='--output .tools/releases/nova-execucao'
# Opcional: resumo versionável sem dados de aplicação; arquivo NOVO
make release-artifacts ARGS='--output .tools/releases/outra-execucao --report docs/evidence/images-nova-execucao.json'
```

Exige Python 3.12+, Docker/Compose local por socket Unix e internet para imagens/base de vulnerabilidades. Não recebe imagem arbitrária nem credenciais de registry de publicação. Constrói os runtimes API e web do checkout; baixa DB e migrator pelos digests de `deploy/images.lock.json`. Dockerfiles/Compose também fixam essas referências. Não inicia banco/serviço da aplicação para escanear e não altera os containers de desenvolvimento.

O Trivy 0.69.3 está fixado por digest. Essa versão foi escolhida após consultar o [advisory do fornecedor sobre o comprometimento do ecossistema](https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23): não usar tags flutuantes, versão 0.69.4 nem imagens 0.69.5/0.69.6 daquele incidente. Não foi adicionado trivy-action/setup-trivy. Isso não substitui verificação de proveniência/assinaturas das ferramentas em uma política completa de release.

A etapa de download da DB acessa somente seu cache e temporário, sem socket/repositório/segredos montados. Os scans posteriores usam `--network none`, arquivo de imagem exportado somente leitura, cache, saída e temporário próprios, usuário do host sem capabilities, filesystem raiz read-only, 1 CPU/1 GiB. O temporário usa disco em `.tools`, pois a extração da DB excedeu o tmpfs inicial de 256 MiB; não reduzir os limites sem ensaio. O cache DB pode ocupar alguns GiB. O processo Docker do harness requer o acesso local já usado no desenvolvimento; o scanner não recebe o socket.

Por imagem gera:

- `api|web|db|migrate.cdx.json`: SBOM CycloneDX com inventário e vulnerabilidades detectadas.
- `api|web|db|migrate.scan.json`: relatório nativo Trivy com severidade, versões detectadas/correções conhecidas e metadados de imagem.
- `manifest.json`: image IDs, referência upstream quando aplicável, versão/data da DB, contagens, achados e SHA-256 dos oito artefatos.

Artefatos completos ficam em `.tools/releases/<execucao>` (ignorados pelo Git). A evidência versionada em `docs/evidence/images-2026-10-25.json` é o resumo do snapshot; não inclui imagens, credenciais ou dados de banco. Para aquela execução, os SBOMs estão em `.tools/releases/6e-scanned`. Guarda os arquivos se precisar auditoria durável; repetir com DB mais nova pode produzir resultados diferentes. Imagens locais são removidas após análise; upstream/cache permanecem para evitar downloads repetidos. Não usar `docker system prune` como rotina do harness.

Gate: zero HIGH/CRITICAL nos quatro runtimes, incluindo achados sem correção disponível. Sem `--ignore-unfixed`, suppressions ou VEX implícito. UNKNOWN/MEDIUM/LOW ficam registrados para triagem; passar esse gate não significa ausência de vulnerabilidades. Inventário ausente ou ferramenta/JSON inválidos falham de forma fechada. Retorno 0: gate aprovado; 2: artefatos completos, gate bloqueado por achados; 1: falha de ferramenta/dados; 130: interrupção. A saída/relatório nunca é sobrescrita.

Os digests upstream são referências imutáveis de manifest/index; o `imageId` de build local identifica sua configuração e **não é** um digest de imagem publicada em registry. Ainda não existem assinatura/attestation, provenance/SLSA ou imagem pública do FlagPilot. Em eventual publicação autorizada, registrar os digests do registry e vincular SBOMs àqueles artefatos exatos. O inventário do web runtime não cobre toda a cadeia npm/build: manter também npm audit, govulncheck e revisão das dependências de build.

## Rollback de binário com schema compatível

```bash
make rollback-rehearsal
make rollback-rehearsal ARGS='--report docs/evidence/rollback-novo-ensaio.json'
```

Reconstrói a fonte local 6B (`2a05bff`) com o mesmo Go/base fixado usado pela candidata atual; nenhuma versão antiga publicada é alegada. Não faz checkout/reset/stash na árvore do usuário. Stack `flagpilot-rollback-*`, senhas privadas temporárias, porta loopback aleatória, DB/volume próprios. Executa anterior → candidata → anterior → candidata com verificação do image ID realmente em execução e readiness a cada troca.

Cria fixture via CLI/API, salva rollout de 37,25% na candidata, revoga token anterior e emite novo token. Antes/depois da reversão compara conteúdo canônico de todas as 11 tabelas em memória. Confere sessão, token novo, rejeição do revogado, mesma avaliação/revisão e escrita após rollback preservada na reaplicação da candidata. O DB/container é mantido; schema 5 continua clean; não executa migration down, restore ou perda de dados. Relatório contém apenas hashes de fontes/imagens, contagens, tempos e booleanos.

É ensaio de reversão da **API** com schema idêntico, não rollback completo de frontend/proxy/DB ou teste de uma versão incompatível. Tempos incluem operações locais de Compose/readiness; não medem indisponibilidade real nem RTO produtivo. Não habilitar tráfego com schema incompatível. Para mudanças futuras usar expand/contract, matriz explícita de versões compatíveis e backup/restore revisados; não presumir que uma migration down seja segura. Restore é outro fluxo e pode perder mudanças posteriores ao snapshot, conforme [recuperação](recovery.md).

Ctrl+C tenta limpar somente containers/volumes/imagens/temporários do projeto próprio. Queda/SIGKILL pode deixar recursos: identificar o prefixo aleatório e conferir labels antes de remover. Não remover `flagpilot-dev` ou seus volumes. Nenhuma rotina real de backup/retention/alerta é criada.

## CI preparada

`.github/workflows/release-evidence.yml` é manual (`workflow_dispatch`), sem login em registry, deploy ou segredos de publicação. Roda guardas, rollback e scans; guarda somente JSONs produzidos (sete dias) mesmo quando o gate bloqueia. Actions fixadas por SHA. Pipeline regular recebe os testes rápidos do gate de inventário. Não foi executado no GitHub nesta etapa; a existência do workflow não prova CI remota aprovada nem autoriza sua publicação/implantação.

Fontes: [SBOM Trivy](https://trivy.dev/docs/latest/guide/supply-chain/sbom/), [release 0.69.3](https://github.com/aquasecurity/trivy/releases/tag/v0.69.3), [upload-artifact](https://github.com/actions/upload-artifact), [ADR-015](../adr/015-release-evidence.md).
