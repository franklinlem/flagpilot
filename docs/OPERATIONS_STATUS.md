# Incremento 6C — recuperação sintética e métricas

Atualização 7A: GitHub privado e homologação externa autorizados, com backup recorrente e monitoramento no host físico. Estado atual em [HOMOLOGATION_STATUS.md](HOMOLOGATION_STATUS.md); pendências abaixo são snapshots das respectivas etapas.

Data: 25/10/2026. Continuidade autorizada após 6B. Escopo local: nenhuma operação sobre dados reais, servidor externo, monitor ou backup produtivo.

## Entregas

- `scripts/recovery.py` / `make recovery-rehearsal`: cria fixture via API/CLI, pg_dump custom privado/temporário, remove origem e verifica seus volumes ausentes, restaura em DB novo e compara todas as 11 tabelas por contagem e hash canônico em memória. Roles do destino têm novas senhas; ACLs/privilégios preservados.
- Antes de iniciar a API restaurada, invalida todas as sessões/tokens e registra evento de recuperação. Testa revogação posterior ao snapshot, credenciais antigas rejeitadas, reemissão, mesma avaliação/revisão/salt e runtime sem DDL. Dump/.env são apagados; relatório só tem tempos/contagens/tamanho/booleanos.
- Cliente oficial Prometheus client_golang 1.25.0 fixado; registry por processo, métricas HTTP de volume/status/duração por padrão de rota, processo/runtime e pool DB. Sem contexto/IDs/chaves de flags/credenciais/IP como labels. Métodos livres viram OTHER, caminhos inexistentes unmatched.
- Listener separado METRICS_ADDR, desativado por padrão no binário. Compose full habilita porta interna 9090 e publica só loopback por METRICS_PORT. API principal não monta métricas. Limites de coleta/HTTP e shutdown cobrem ambos os listeners. Novas opções em .env.example, sem segredo novo nem migration (schema 5).
- CI configurada para rodar ensaio de recuperação sintético além do smoke e browsers existentes; não houve execução no GitHub.

## Evidências

`make ci` passou: Go/race/vet, tipos/lint/OpenAPI, PostgreSQL real, 21 testes frontend, 16 SDK e builds. Novos testes verificam configuração opt-in/endereço/porta, separação de listeners, registry independente e ausência de dados de request nas métricas. Não há testes que dependam de valores reais do notebook ou credenciais do usuário.

Host Docker remoto recusado antes de criar recursos; contexto local exigido e variáveis de credenciais/projeto herdadas ignoradas pelo harness.

Ensaio de recuperação aprovado e [relatório JSON](evidence/recovery-2026-10-25.json) registrado: tabelas completas preservadas, origem destruída antes de restaurar, invalidação auditada antes do startup, avaliação estável após reemissão e runtime sem DDL. A fixture pequena não mede RPO/RTO produtivo; tempos do arquivo são somente daquele ensaio.

Smoke atualizado aprovado: scrape real na porta interna, pool máximo 5, API principal retornando 404 para métricas, falha de bind encerra segunda API, SIGINT com saída zero/reinício, liveness com DB indisponível/readiness 503 e scrape ainda disponível. Todos em stacks descartáveis próprias.

Seis fluxos Chromium passaram novamente em 6C contra a API instrumentada. A matriz Firefox validada em 6B permanece como evidência anterior, sem nova execução local em 6C (nenhuma mudança de comportamento do frontend).

Govulncheck executado após adicionar o cliente Prometheus: zero vulnerabilidades alcançáveis e zero nos pacotes importados; o scanner informa uma vulnerabilidade em módulo requerido não chamado. Não é afirmação de ausência absoluta de vulnerabilidades; scans de imagens/SBOM e gate de release continuam pendentes. npm não teve dependências alteradas nesta etapa.

## Limites e continuidade

Ver [recuperação](runbooks/recovery.md), [observabilidade](runbooks/observability.md), [ADR-013](adr/013-recovery-observability.md) e [critérios do MVP](MVP_READINESS.md). Ensaio pontual não configura backup recorrente, criptografia de backups reais, retenção, cópia externa ou alerta. Não foi instalado/configurado coletor Prometheus/Grafana/Uptime Kuma. Nenhum deploy externo, RPO/RTO produtivo ou benchmark/SLO é reportado como concluído.

Continuidade 6D: medição local de capacidade concluída; resultados e limites em [CAPACITY_STATUS.md](CAPACITY_STATUS.md). Continuidade 6E: artefatos/scans e rollback compatível entregues, mas gate de imagens bloqueado; ver [RELEASE_STATUS.md](RELEASE_STATUS.md). Próximo incremento: remediação/triagem de imagens. Coleta/alertas e recuperação operacional real continuam pendentes antes de homologação. Qualquer mudança em VPS/servidores/domínios continua exigindo autorização específica.

Verificações finais: containers de desenvolvimento reconstruídos e saudáveis, volume preservado; readiness/frontend/métricas HTTP 200, /metrics na API principal HTTP 404. sqlc/OpenAPI regenerados sem diferenças e Gitleaks staged aprovado. Nenhum dado real foi usado pelo ensaio, nenhuma credencial real foi emitida automaticamente e nenhum servidor externo foi modificado.

Continuidade 6F: achados HIGH/CRITICAL de imagens corrigidos e gate aprovado; PostgreSQL de desenvolvimento atualizado dentro da major 16 com dados preservados. Ver [SECURITY_REMEDIATION_STATUS.md](SECURITY_REMEDIATION_STATUS.md). Backup recorrente, monitor externo e implantação continuam pendentes.
