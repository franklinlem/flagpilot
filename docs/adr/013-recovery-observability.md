# ADR-013 — Recuperação sintética e listener de métricas

Data: 25/10/2026. Estado: implementada localmente em 6C.

## Recuperação

Ensaio reproduzível separado de desenvolvimento/produção, sem parâmetros de destino ou dump externo. Exige Docker local por socket Unix, recusa contextos/hosts remotos e isola credenciais/variáveis Compose herdadas do shell. Usa pg_dump custom completo, remove a origem, restaura em DB vazio com --no-owner/--single-transaction/--exit-on-error e compara os dados de todas as tabelas antes de ajustes operacionais. Roles não pertencem ao dump: destino provisiona migrador/runtime com novas senhas e mantém grants do arquivo. Backup comprimido não é criptografado; o arquivo é somente sintético, temporário e privado. Relatório público local contém contagens/tempos/booleanos, nunca linhas ou credenciais.

Uma restauração volta ao snapshot e pode ressuscitar revogações posteriores. A fixture provoca esse caso; a política do ensaio invalida todas as sessões/api_tokens e audita antes de abrir a API restaurada. Depois autentica novamente e reemite uma credencial para validar avaliação estável e privilégios. Não há backup recorrente, retenção, cópia externa ou RPO/RTO produtivo comprovados. A autorização atual não inclui restauração real ou implantação externa.

## Observabilidade

Uma dependência direta nova, cliente oficial Prometheus client_golang 1.25.0, fixada e com registry por instância. Métricas HTTP agregadas por método conhecido/padrão de rota/status e histogramas de duração; runtime/processo e gauges de pool sem labels de domínio. Sem contexto, usuário, token, chave de flag, URL livre ou IDs como rótulos; desconhecidos viram unmatched/OTHER. Logs continuam com request ID e padrão de rota, sem headers/body/query.

METRICS_ADDR é opt-in e vazio por padrão. Listener separado /metrics, sem autenticação de aplicação, com limite de scrapes/timeout/headers e acesso restrito à rede confiável. API principal continua 404 para métricas. Compose full habilita escuta interna e publicação somente em loopback, porta externa METRICS_PORT (9090 padrão). Endereço deve ser IP privado/loopback/unspecified explicitamente configurado e porta distinta; nunca configurar ingress público. Shutdown cobre ambos os servidores, incluindo falha de bind.

Não adicionar stack de monitoramento ao notebook. Coletor/alertas, dimensionamento, benchmark e segurança do destino são próximos escopos. Nenhuma migration: schema 5 preservado. Novas variáveis documentadas em .env.example; nenhum segredo novo de servidor.

## Referências

- [pg_dump 16](https://www.postgresql.org/docs/16/app-pgdump.html) e [pg_restore 16](https://www.postgresql.org/docs/16/app-pgrestore.html).
- [Release client_golang 1.25.0](https://github.com/prometheus/client_golang/releases/tag/v1.25.0) e [instrumentação oficial Go](https://prometheus.io/docs/guides/go-application/).
