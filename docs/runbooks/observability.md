# Observabilidade mínima local

API mantém logs JSON com request ID, método, padrão de rota, status e duração. Não registrar URL/query/body/Authorization/cookie/bucketingKey. O request ID vem do servidor e acompanha erros. Health/readiness permanecem na API principal.

## Consultar métricas

No Compose de desenvolvimento, o listener interno 9090 é publicado somente em 127.0.0.1:9090, configurável por METRICS_PORT. Após make dev-containers:

```bash
curl --fail --silent --show-error http://127.0.0.1:9090/metrics
```

Ao rodar Go no host, METRICS_ADDR vazio (padrão) desativa o listener. Para habilitar no host, configure METRICS_ADDR=127.0.0.1:9090 no ambiente. Esse valor é endereço de escuta, não URL. Configuração exige IP loopback/privado/explícito de container e porta diferente da API; não aceitar hostname ou IP público. 0.0.0.0/:: são permitidos somente por configuração explícita, para redes de container, e exigem isolamento de rede.

O endpoint NÃO está montado na API principal: /metrics e /api/v1/metrics em 8080 retornam 404. Não adicione rota de Traefik/Coolify ou proxy web para a porta de métricas. O listener não usa sessão/token e depende da rede confiável; não publique na borda. Não foi instalado Prometheus, Grafana, Uptime Kuma ou outro serviço de coleta/alerta nesta etapa.

## Métricas entregues

| Família | Significado |
|---|---|
| flagpilot_http_requests_total | Contagem de requests concluídos por method, padrão de route e status |
| flagpilot_http_request_duration_seconds | Histograma de duração por method/route; buckets em segundos, incluindo 0.1 |
| flagpilot_db_pool_connections | Conexões atuais do pool |
| flagpilot_db_pool_acquired_connections | Conexões em uso |
| flagpilot_db_pool_idle_connections | Conexões ociosas |
| flagpilot_db_pool_max_connections | Limite configurado (5 atualmente) |
| go_* / process_* | Medidas padrão de runtime/processo do cliente oficial, conforme plataforma |

Rótulos HTTP nunca usam UUID, chave de flag, token, usuário, ambiente de domínio, IP ou contexto. Caminhos inexistentes viram unmatched; métodos não previstos viram OTHER. Não há label de motivo/identidade da avaliação. O registry é exclusivo da instância e seus contadores reiniciam com o processo. O scrape não entra nos contadores da API, e estatísticas do pool não consultam o banco nem comprovam saúde do DB.

O listener aceita até dois scrapes simultâneos, timeout de coleta 5 s, limites de headers e shutdown coordenado com a API. Se uma das portas falhar, a aplicação encerra os listeners em vez de manter uma inicialização parcial. Coleta de métricas não substitui /api/v1/health/ready.

## Verificar localmente

`make ci` verifica configuração opt-in, privacidade/agregação de rotas/métodos e separação dos listeners. `python3 scripts/smoke.py` confere métricas reais em porta loopback efêmera, API principal sem endpoint de métricas, shutdown/reinício e scrape mesmo com DB indisponível, em stack descartável.

Ainda faltam coleta/retensão, dashboards, alertas, limites de recursos do destino e benchmark de capacidade. Não há SLO p95 comprovado nem monitor externo configurado/verificado. O cliente oficial está fixado em client_golang 1.25.0 com dependências no go.sum; [guia oficial de instrumentação Go](https://prometheus.io/docs/guides/go-application/).
