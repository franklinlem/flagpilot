# Medição local de capacidade

Incremento 6D. Não direcionar o ensaio para desenvolvimento persistente, VPS ou produção. O harness não aceita URL, token, volume ou arquivo de configuração externo. Exige Docker Engine em Linux, socket Unix e contexto local. Usa k6 1.6.1 fixado por digest, sem instalação global ou dependência na aplicação.

```bash
make load-rehearsal
# Opcional: salvar evidência agregada em arquivo NOVO
make load-rehearsal ARGS='--report docs/evidence/load-nova-medicao.json'
```

O download inicial exige internet para imagens; a carga HTTP é exclusivamente local. Cada execução constrói API, cria projeto Compose aleatório, senhas temporárias privadas, DB próprio e fixture via CLI/API: organização, projeto, ambientes padrão, uma flag habilitada a 100% e um token sintético. Nenhum segredo é passado na linha de comando ou incluído no relatório. Credenciais ficam em memória/arquivo temporário 0600; stdout/stderr de ferramentas são capturados. `finally` remove apenas container k6, serviços e volumes do projeto descartável e arquivos temporários. SIGKILL/queda do host podem impedir limpeza: localizar somente os projetos com prefixo `flagpilot-load-` e conferir antes de remover seus recursos. Não usar `docker system prune` nem remover `flagpilot-dev`.

## Método

- API: 1 CPU, 256 MiB; DB: 1 CPU, 512 MiB; gerador: 1 CPU, 256 MiB. Pool máximo 5, limites originais de avaliação ativos (100/s token, 120/s peer, 600/s global). Host compartilhado com a stack de desenvolvimento e outros processos; cotas não reservam núcleos exclusivos.
- Dez requests de aquecimento separados da medição; espera de 1,2 s entre cenários. Cenários sequenciais 10, 50, 100 e 150 req/s, cada um com 30 s, 20 VUs pré-alocados/máximos, timeout de request de 5 s e encerramento de até 10 s. Sem retries, cache SDK ou pausas artificiais por iteração.
- Executor `constant-arrival-rate`: inicia iterações independentemente da duração da resposta. Uma avaliação HTTP por iteração, conexão reutilizável, contexts sintéticos variados; body/status/flag/ambiente/valor/reason verificados. O caso 150/s verifica rejeição sob sobrecarga do limitador, não o máximo de throughput do banco.
- Relatório separa respostas 200 válidas, 429 e erros (rede, timeout, outros status ou corpo incorreto). `http_req_failed` do k6 inclui 429; não confundir esse agregado com erro interno. Requests descartados pelo gerador (`dropped_iterations`) permanecem visíveis e reprovam a execução. Um 429 não é contado como sucesso nem prova de capacidade.
- p50 (`med`), p95 e p99 em `evaluation_success_wall_ms`: tempo de parede do cliente em milissegundos inteiros, incluindo request/resposta e estabelecimento quando necessário, somente respostas 200 válidas. `evaluation_wall_ms` inclui todos os resultados; `http_req_duration` usa a medição nativa do k6. Não medir apenas latências rápidas de rejeição.
- `achievedRequestsPerScheduledSecond` divide requests realizados pelos 30 s programados. `wallSecondsIncludingK6Startup` inclui startup/shutdown do container/gerador, portanto não é o denominador desse throughput. Registrar descartes, erros e proporção de sucesso junto à taxa oferecida.
- `docker stats` e scrape Prometheus coletados periodicamente durante cada cenário: CPU/memória de API/DB e conexões adquiridas do pool. São amostras esparsas (intervalo 2 s após cada coleta), sem captura de picos entre coletas nem garantia de ausência de saturação. CPU 100% equivale a um núcleo; memória do Docker pode descontar cache. A coleta também tem custo no host. Não há perfil de CPU, fila interna ou medição de CPU do gerador; descartes indicam execução insuficiente e invalidam o cenário.

## Interpretação e limites

Em 100 req/s, jitter e janela fixa do limitador podem produzir 429 mesmo sem falta de CPU/DB; não elevar/desabilitar limites para melhorar o benchmark. 10/50 req/s devem completar sem throttling. 150 req/s deve produzir rejeições controladas. Falha de validação, descarte do gerador ou falha de coleta reprova o ensaio e mantém relatório, se solicitado. A latência p95 é evidência, sem threshold rígido no CI compartilhado.

O fixture é pequeno: uma flag habilitada a 100%, um ambiente usado e um token. Não cobre batch, catálogo grande, rollout parcial, múltiplos tenants/tokens simultâneos, escrita administrativa concorrente, soak prolongado, rede externa, proxy/TLS ou VPS. O limite por token/peer impede inferir capacidade máxima do serviço; 30 s não demonstra SLO sustentado ou comportamento sob falha. Registrar hardware, imagem, revisão e cotas ao comparar execuções. Para homologação, definir carga/dataset/concorrência/tempo/isolamento com inventário e autorização próprios.

## Referências

[Executor de taxa constante](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/), [métricas e dropped_iterations](https://grafana.com/docs/k6/latest/using-k6/metrics/reference/) e [release k6 1.6.1](https://github.com/grafana/k6/releases/tag/v1.6.1). Ver [estado de capacidade](../CAPACITY_STATUS.md).
