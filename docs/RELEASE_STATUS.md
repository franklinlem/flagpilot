# Estado de release — atualização 6F

**Gate de imagens agora aprovado:** zero HIGH/CRITICAL nos quatro runtimes corrigidos. Web/DB/migrator sem achados; API com dois UNKNOWN de OpenPGP não importado. Correções, evidências e limites em [SECURITY_REMEDIATION_STATUS.md](SECURITY_REMEDIATION_STATUS.md). Release/produção continuam pendentes dos demais critérios.

O registro abaixo é o snapshot histórico bloqueado de 6E, preservado para comparação.

## Incremento 6E — imagens, SBOM e rollback local

Data: 25/10/2026. **Gate de imagens bloqueado. Não é release aprovada.** Continuidade autorizada após 6D; nenhuma publicação, deploy ou alteração de servidores/dados de desenvolvimento.

## Entregas

- Bases dos Dockerfiles e imagens DB/migrator do Compose fixadas por digest, mesmas versões. Referências em `deploy/images.lock.json`; pinning não elimina vulnerabilidades existentes.
- `make release-artifacts`: builds dos runtimes API/web, análise desses e de PostgreSQL/migrate; Trivy 0.69.3 fixado por digest, SBOM CycloneDX + JSON nativo, manifest de hashes/IDs/datas/achados. Scanner sem socket Docker, sem segredos/repositório montados; análise dos arquivos de imagens com rede desligada. Arquivos completos ignorados em `.tools/releases/6e-scanned`; resumo versionado em [images-2026-10-25.json](evidence/images-2026-10-25.json).
- Gate exige inventário válido e zero HIGH/CRITICAL, incluindo unfixed; nenhuma supressão criada. UNKNOWN e demais severidades não são ignoradas no relatório. `make rollback-rehearsal` cria stack própria e testa anterior/candidata/reversão/reaplicação com schema 5.
- Workflow manual de evidências preparado, sem push/deploy/login em registry; artefatos JSON com retenção de sete dias e actions fixadas por SHA. Não executado no GitHub. CI regular inclui regressões rápidas do gate.

## Scan observado

Trivy DB atualizada em 25/10/2026 12:57 UTC, baixada às 15:02 UTC. Scanner 0.69.3. Contagens são ocorrências reportadas por pacote/binário, não número de CVEs únicas nem prova de exploração da aplicação.

| Imagem | Componentes SBOM | CRITICAL | HIGH | MEDIUM | LOW | UNKNOWN | Gate |
|---|---:|---:|---:|---:|---:|---:|---|
| API scratch/Go | 35 | 0 | 0 | 0 | 0 | 2 | Passou o limiar; UNKNOWN em triagem |
| Web NGINX 1.28.2 Alpine | 72 | 3 | 67 | 104 | 53 | 4 | Bloqueado |
| PostgreSQL 16.13 Alpine | 51 | 1 | 36 | 47 | 35 | 3 | Bloqueado |
| migrate 4.20.1 | 124 | 0 | 13 | 17 | 3 | 3 | Bloqueado |

Web: NGINX e pacotes de sistema (OpenSSL/curl/expat/musl e outros). DB: bibliotecas Alpine e stdlib do binário auxiliar Go incluído na imagem. Migrator: stdlib Go e bibliotecas/drivers empacotados, incluindo dependências sem versão corrigida informada. API: duas ocorrências UNKNOWN de x/crypto v0.58.0 em admin/server. Arquivos contêm IDs, versões e URLs upstream retornadas pelo scanner; validar cada aviso e seu alcance antes de concluir risco/mitigação. Não houve exceção por driver não usado nem por componente auxiliar.

Resultado do comando: exit 2, artefatos completos, **116 HIGH e 4 CRITICAL** entre os runtimes. O gate foi bloqueado conforme planejado; não é falha silenciosa da ferramenta. Foram conferidos os quatro inventários e os oito hashes dos artefatos. O download inicial falhou por espaço no tmpfs de 256 MiB; foi substituído por temporário privado em disco, mantendo scanner sem acesso ao socket/segredos. Base/cache ocupa cerca de 1,6 GiB neste snapshot.

## Rollback observado

[Relatório](evidence/rollback-2026-10-25.json): fonte anterior 2a05bff (6B), candidata baseada em 7feadec com Dockerfile fixado em árvore de trabalho; imagens diferentes e image ID em execução confirmado a cada troca. Fonte anterior reconstruída com a mesma base Go fixada atual, sem alegar imagem histórica de release. Todos usam schema 5.

Anterior → candidata → anterior → candidata passou. Conteúdo das 11 tabelas preservado na reversão, DB/container mantido, mesma avaliação percentual/revisão, sessão válida, token emitido pela candidata aceito e token revogado ainda rejeitado. Escrita feita após rollback permaneceu no reupgrade. Nenhum migration down/restore foi usado. Trocas/readiness levaram aproximadamente 5 s cada; não equivalem a RTO ou indisponibilidade produtiva. Containers, volume e tags do ensaio foram removidos.

## Verificações e limites

Três regressões automatizadas do gate passaram (inventário ausente, HIGH/CRITICAL unfixed, inventário limpo/LOW), Python compilado. Docker remoto, arquivos/saídas existentes recusados antes de provisionamento. Sem código de negócio, schema ou dependências Go/npm alterados. Scans completos e rollback foram reais, em recursos locais próprios; nenhuma aceitação visual nova é necessária para estes scripts. Smoke após pinning registrado no fechamento abaixo. Não houve execução de CI remota.

Próximo incremento: remediar/triagem dos achados, com atualização controlada das imagens de base e, se necessário, build restrito do migrator; manter major PostgreSQL 16 e validar cada mudança em DB sintético antes de qualquer atualização do desenvolvimento. Repetir scans/SBOM após as correções; não ampliar allowlists para obter gate verde. UNKNOWN exige análise separada. Sem release até revisão da política/achados.

Backup recorrente real, coletor/alertas, TLS/proxy/VPS, licença/publicação, assinaturas/provenance, digests do registry e rollback de schema incompatível permanecem pendentes. Nenhum SLO/RPO/RTO produtivo foi demonstrado. Uso e fontes em [runbook](runbooks/release.md), decisão em [ADR-015](adr/015-release-evidence.md).

Fechamento local: smoke após fixação de digests aprovado (live/ready, métricas, SIGINT/reinício e falha real do DB). Stack/volume do ensaio removidos; desenvolvimento continua saudável e não foi reconstruído nesta etapa. Inventários e hashes dos oito arquivos conferidos contra o resumo usando o gate final. Suítes de negócio e browser não foram repetidas, pois não houve mudança de comportamento/contrato/dependências da aplicação; build dos runtimes e smoke cobriram a alteração de referência das imagens.
