# FlagPilot — critérios locais do MVP

Atualização 7A: GitHub privado e homologação externa autorizados, com backup recorrente e monitoramento no host físico. Estado atual em [HOMOLOGATION_STATUS.md](HOMOLOGATION_STATUS.md); pendências abaixo são snapshots das respectivas etapas.

Atualizado em 25/10/2026, incremento 6F. Este documento registra o que foi verificado localmente e o que ainda falta; não declara release 1.0, homologação ou aptidão para produção. Franklin confirmou manualmente os filtros e a visualização do histórico em 6A e autorizou a continuidade.

## Evidências atuais

| Área | Estado e evidência |
|---|---|
| Autenticação/organizações/papéis | Implementados, com testes de sessão, CSRF, isolamento, último Owner e auditoria. Aceitações manuais anteriores registradas nos status de cada etapa. |
| Projetos/ambientes | Criação/consulta e presets atômicos implementados. Edição/exclusão continuam fora dos incrementos atuais até definir retenção/concorrência. |
| Flags/rollout/tokens | Catálogo, arquivamento, estado/percentual por ambiente, If-Match, produção explícita, tokens com expiração/revogação e avaliação single/batch verificados automaticamente; fluxos manuais relatados nos status. |
| SDK backend | Pacote privado 0.1.0 ESM/CJS, tipos, cache/timeout/fallback e 16 testes; integração real em stack descartável. Aceitação manual em aplicação consumidora não foi relatada. |
| Auditoria/filtros | Somente leitura no escopo autorizado, projeção de campos conhecidos, paginação temporal e filtros no banco; aceitação manual de Franklin em 6A. |
| Frontend/contrato | `make ci` passou: tipos, ESLint, Spectral, 21 testes frontend, Go com race detector, integração PostgreSQL e builds. Tipos/queries gerados acompanham o contrato. |
| Navegadores | Em 6B, seis fluxos por execução em Chromium e Firefox headless passaram (12 no total); em 6C, seis fluxos Chromium foram repetidos e passaram, com stacks/credenciais/volumes próprios. Os limites da aplicação permanecem ativos. CI configurada para ambos, ainda sem execução no GitHub. |
| Acessibilidade e navegação | Labels, foco nos títulos, título da aba, skip link por Enter, details/summary por teclado, retorno ao topo, estados de erro/vazio/conflito e largura mobile exercitados. Desktop padrão 1280 px; verificações de overflow em 375/390 px nos cenários de histórico/projetos. |
| Recuperação de conexão | Regressão de deadline com sinal de consulta reproduzida e corrigida: timeout do painel permanece em 8 s e cancelamento também é respeitado. E2E injeta 503 na sessão/histórico e verifica recuperação sem login falso. |
| Resiliência dos containers | `scripts/smoke.py` aprovado em stack sintética: live/ready, SIGINT com saída zero, reinício e perda real de DB (liveness responde/readiness 503). Volume de desenvolvimento não foi usado para esses testes. |
| Migrations e privilégios | Schema 5 mantido, nenhuma migration em 6B/6C/6D/6E/6F. Upgrade com dados preservados, runtime sem DDL e escrita/auditoria/revisão atômicas verificados nas suítes PostgreSQL. |
| Dependências | Cliente oficial Prometheus 1.25.0 fixado em 6C. Govulncheck atual sem vulnerabilidades alcançáveis/importadas; uma ocorrência em módulo requerido não chamado. Scans/SBOM repetidos após correções 6F: zero HIGH/CRITICAL nos quatro runtimes, dois UNKNOWN de OpenPGP não importado na API; ver SECURITY_REMEDIATION_STATUS.md. |
| Recuperação sintética (6C) | Dump/restore aprovado com origem destruída, comparação das 11 tabelas, roles com novas senhas, invalidação auditada de sessões/tokens antes do startup e avaliação estável após reemissão. Relatório em docs/evidence; não equivale a rotina real de backup. |
| Métricas (6C) | Listener Prometheus separado, opt-in no binário, somente loopback no Compose local. Volume/status/duração HTTP, runtime/processo e pool; testes de privacidade/isolamento e smoke aprovados. |
| Capacidade local (6D) | k6 com fixture isolada e cotas, 9.301 requests/zero erros internos ou de validação/zero descartes; rejeições esperadas do limitador separadas. Evidência e condições em CAPACITY_STATUS.md. |
| Evidências de release (6E) | SBOMs/hashes e achados registrados; guardas e três regressões do gate aprovadas. Reversão/reaplicação da API com schema 5 preservou todas as tabelas/sessão/revogação/escritas. Gate bloqueado no snapshot 6E, corrigido em 6F; workflow manual preparado sem execução remota. |
| Correções 6F | PostgreSQL 16.15 com gosu recompilado, NGINX 1.30.6/APKs corrigidos e migrator PostgreSQL-only. make ci, Chromium (6 fluxos), recuperação/upgrade e rollback aprovados; desenvolvimento atualizado com cópia privada prévia e conteúdo preservado. |

## Critérios ainda abertos

| Critério | Trabalho necessário antes de concluir |
|---|---|
| Backup/restore operacional | Ensaio sintético concluído em 6C; backup recorrente real, criptografia/chaves, retenção, cópia externa e alertas ainda não configurados. Recuperação real segue pendente de inventário/isolamento e autorização. |
| RPO/RTO | Alvos propostos RPO ≤24 h e RTO ≤4 h continuam sem evidência de recuperação; não são garantias. |
| Métricas/alertas | Health/readiness e logs JSON existem. Endpoint Prometheus entregue em listener separado; coletor, dashboards, alertas e monitor externo ainda pendentes. Configuração de monitor do FlagPilot no Uptime Kuma não foi executada/verificada nesta etapa. |
| Performance/capacidade | Ensaio local 6D de 10/50/100 req/s e 150/s com limites ativos: p50/p95/p99, erros/rejeições/descartes e recursos registrados em CAPACITY_STATUS.md. p95 de sucesso 7–13 ms para fixture pequena; 15 respostas 429 em 100/s. SLO sustentado, saturação máxima e sizing da VPS permanecem sem comprovação. |
| Acessibilidade completa | Revisão WCAG 2.2 AA, leitor de tela, zoom, matriz completa de viewports/estados e aceitação visual humana em todos os browsers permanecem pendentes. Testes atuais não equivalem a certificação. |
| Segurança/release | 6F corrigiu os runtimes: gate aprovado sem HIGH/CRITICAL; dois UNKNOWN de módulo OpenPGP ausente dos imports/símbolos da API permanecem visíveis. Proveniência/assinaturas/digests de registry, revisão de licença/publicação e rollback com schema incompatível continuam pendentes. Não equivale a avaliação ASVS integral. |
| Política de dados | Retenção/limpeza automática de auditoria/sessões, exportação e identidade histórica nominal dos atores ainda sem entrega. Auditoria identifica por ID/rótulo e não restaura snapshots. |
| Browser consumidor | Aplicação pública usa BFF autorizado; não existe token publicável ou adapter público genérico. SDK secreto continua exclusivo para backend. |
| Infraestrutura externa | Inventário/capacidade, isolamento homologação/produção, credenciais, proxy confiável/rate limit, TLS/DB, DNS, Coolify/Traefik e testes externos ainda exigem planejamento e autorização específica. Nada foi alterado em VPS/servidor físico/domínios. |
| Publicação/licença | Licença, nome público do pacote, repositório remoto e publicação de imagens/npm/releases permanecem pendentes. Versionamento atual é local em Git. |

## Reprodução

Na raiz do projeto:

```bash
make ci
make recovery-rehearsal
make load-rehearsal
make rollback-rehearsal
# Retorno 2 se HIGH/CRITICAL; diretório novo para cada execução
make release-artifacts ARGS='--output .tools/releases/nova-execucao'
python3 scripts/smoke.py
PLAYWRIGHT_BROWSERS_PATH="$PWD/.tools/playwright" npx playwright install chromium firefox
python3 scripts/auth-e2e.py --browser chromium
python3 scripts/auth-e2e.py --browser firefox
```

Executar os harnesses de browser separadamente: cada execução cria sua stack e depois remove apenas os recursos descartáveis próprios. Não substituir pelo volume de desenvolvimento ou credenciais reais. Instalação de dependências de sistema dos browsers deve seguir o ambiente do notebook; CI usa --with-deps no runner Ubuntu próprio.

Próximo incremento proposto: preparação operacional e de release, incluindo licença/proveniência, backup recorrente/coleta/alertas revisáveis. Remediação de imagens concluída em 6F; ver SECURITY_REMEDIATION_STATUS.md. Nenhuma configuração externa ou publicação autorizada implicitamente. Capacidade local medida em 6D (CAPACITY_STATUS.md); sizing externo, coletor/alertas e backup recorrente permanecem pendentes. Ensaio sintético e listener de métricas foram entregues em 6C; ver OPERATIONS_STATUS.md. Preparar esses artefatos não autoriza deploy nem configurações externas. Ver [DASHBOARD_STATUS.md](DASHBOARD_STATUS.md) e [runbook do painel](runbooks/dashboard.md).
