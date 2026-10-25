# FlagPilot — critérios locais do MVP

Atualizado em 25/10/2026, incremento 6B. Este documento registra o que foi verificado localmente e o que ainda falta; não declara release 1.0, homologação ou aptidão para produção. Franklin confirmou manualmente os filtros e a visualização do histórico em 6A e autorizou a continuidade.

## Evidências atuais

| Área | Estado e evidência |
|---|---|
| Autenticação/organizações/papéis | Implementados, com testes de sessão, CSRF, isolamento, último Owner e auditoria. Aceitações manuais anteriores registradas nos status de cada etapa. |
| Projetos/ambientes | Criação/consulta e presets atômicos implementados. Edição/exclusão continuam fora dos incrementos atuais até definir retenção/concorrência. |
| Flags/rollout/tokens | Catálogo, arquivamento, estado/percentual por ambiente, If-Match, produção explícita, tokens com expiração/revogação e avaliação single/batch verificados automaticamente; fluxos manuais relatados nos status. |
| SDK backend | Pacote privado 0.1.0 ESM/CJS, tipos, cache/timeout/fallback e 16 testes; integração real em stack descartável. Aceitação manual em aplicação consumidora não foi relatada. |
| Auditoria/filtros | Somente leitura no escopo autorizado, projeção de campos conhecidos, paginação temporal e filtros no banco; aceitação manual de Franklin em 6A. |
| Frontend/contrato | `make ci` passou: tipos, ESLint, Spectral, 21 testes frontend, Go com race detector, integração PostgreSQL e builds. Tipos/queries gerados acompanham o contrato. |
| Navegadores | Seis fluxos por execução em Chromium e Firefox headless passaram (12 no total), com stacks/credenciais/volumes próprios. Os limites da aplicação permanecem ativos. CI configurada para ambos, ainda sem execução no GitHub. |
| Acessibilidade e navegação | Labels, foco nos títulos, título da aba, skip link por Enter, details/summary por teclado, retorno ao topo, estados de erro/vazio/conflito e largura mobile exercitados. Desktop padrão 1280 px; verificações de overflow em 375/390 px nos cenários de histórico/projetos. |
| Recuperação de conexão | Regressão de deadline com sinal de consulta reproduzida e corrigida: timeout do painel permanece em 8 s e cancelamento também é respeitado. E2E injeta 503 na sessão/histórico e verifica recuperação sem login falso. |
| Resiliência dos containers | `scripts/smoke.py` aprovado em stack sintética: live/ready, SIGINT com saída zero, reinício e perda real de DB (liveness responde/readiness 503). Volume de desenvolvimento não foi usado para esses testes. |
| Migrations e privilégios | Schema 5 mantido, nenhuma migration em 6B. Upgrade com dados preservados, runtime sem DDL e escrita/auditoria/revisão atômicas verificados nas suítes PostgreSQL. |
| Dependências | Sem nova dependência do produto em 6B. Firefox baixado em .tools somente para testes. Lockfiles mantidos; scans SCA/imagens não foram repetidos neste incremento e não são presumidos atuais. |

## Critérios ainda abertos

| Critério | Trabalho necessário antes de concluir |
|---|---|
| Backup/restore | Ensaio reproduzível em banco descartável, com registros de integridade/duração e plano de perda de dados. Backup recorrente, retenção, cópia externa e proteção dos segredos ainda não configurados para FlagPilot. |
| RPO/RTO | Alvos propostos RPO ≤24 h e RTO ≤4 h continuam sem evidência de recuperação; não são garantias. |
| Métricas/alertas | Health/readiness e logs JSON existem. Endpoint Prometheus, dashboards, alertas e monitor externo ainda pendentes. Configuração de monitor do FlagPilot no Uptime Kuma não foi executada/verificada nesta etapa. |
| Performance/capacidade | Ambiente de referência e benchmark p50/p95/p99/erros/saturação não executados. SLO p95 <100 ms não foi demonstrado; rate limit não é prova de capacidade. |
| Acessibilidade completa | Revisão WCAG 2.2 AA, leitor de tela, zoom, matriz completa de viewports/estados e aceitação visual humana em todos os browsers permanecem pendentes. Testes atuais não equivalem a certificação. |
| Segurança/release | Repetir SCA/scan de imagens e verificar política de vulnerabilidades; gerar SBOM, fixar digests de imagens de release e ensaiar rollback com recuperação. Testes atuais não equivalem a avaliação ASVS integral. |
| Política de dados | Retenção/limpeza automática de auditoria/sessões, exportação e identidade histórica nominal dos atores ainda sem entrega. Auditoria identifica por ID/rótulo e não restaura snapshots. |
| Browser consumidor | Aplicação pública usa BFF autorizado; não existe token publicável ou adapter público genérico. SDK secreto continua exclusivo para backend. |
| Infraestrutura externa | Inventário/capacidade, isolamento homologação/produção, credenciais, proxy confiável/rate limit, TLS/DB, DNS, Coolify/Traefik e testes externos ainda exigem planejamento e autorização específica. Nada foi alterado em VPS/servidor físico/domínios. |
| Publicação/licença | Licença, nome público do pacote, repositório remoto e publicação de imagens/npm/releases permanecem pendentes. Versionamento atual é local em Git. |

## Reprodução

Na raiz do projeto:

```bash
make ci
python3 scripts/smoke.py
PLAYWRIGHT_BROWSERS_PATH="$PWD/.tools/playwright" npx playwright install chromium firefox
python3 scripts/auth-e2e.py --browser chromium
python3 scripts/auth-e2e.py --browser firefox
```

Executar os harnesses de browser separadamente: cada execução cria sua stack e depois remove apenas os recursos descartáveis próprios. Não substituir pelo volume de desenvolvimento ou credenciais reais. Instalação de dependências de sistema dos browsers deve seguir o ambiente do notebook; CI usa --with-deps no runner Ubuntu próprio.

Próximo incremento proposto: ensaio **local** de backup/restore, seguido da observabilidade mínima e medição. Preparar esses artefatos não autoriza deploy nem configurações externas. Ver [DASHBOARD_STATUS.md](DASHBOARD_STATUS.md) e [runbook do painel](runbooks/dashboard.md).
