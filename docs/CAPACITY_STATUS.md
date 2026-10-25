# Incremento 6D — capacidade local de avaliação

Data: 25/10/2026. Continuidade autorizada após 6C. Carga somente sintética, em stack descartável própria; nenhum deploy ou alteração de dados de desenvolvimento/infraestrutura externa.

## Entregas e método

`scripts/load.py`, `scripts/load/evaluation.js`, `deploy/compose.load.yml` e `make load-rehearsal`: fixture criada pela CLI/API, cenários k6 de taxa de chegada constante 10/50/100 req/s e sobrecarga controlada de 150 req/s. Imagem k6 1.6.1 fixada por digest, sem dependência nova da aplicação. Cotas: API 1 CPU/256 MiB, DB 1 CPU/512 MiB, gerador 1 CPU/256 MiB. Pool 5 e limites token 100/s, peer 120/s, global 600/s preservados. Cada cenário dura 30 s, com aquecimento prévio e 20 VUs; sem cache/retry SDK. Validação de status/corpo a cada sucesso; 429, erros e descartes separados.

Máquina de referência deste ensaio: Intel i3-3110M 2,40 GHz, 4 CPUs lógicas, cerca de 7,7 GiB RAM, kernel Debian 6.12.113+deb13-amd64. Host compartilhado com desenvolvimento/processos do usuário, sem reserva exclusiva de CPU. Load average antes: 2,57/2,76/2,77. API construída a partir de f9d958c (nenhum código da API mudou em 6D); image ID, Docker/k6/kernel e condições completos no [relatório JSON agregado](evidence/load-2026-10-25.json).

## Resultado observado

Latência em ms de parede do cliente, somente respostas 200 com corpo correto. p50/p95/p99 não incluem rejeições; as latências de todos os resultados também estão no JSON.

| Oferta req/s | Requests | 200 válidos | 429 | Erros | Descartes | p50 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 10 | 300 | 300 | 0 | 0 | 0 | 9 | 13 | 19,02 |
| 50 | 1.500 | 1.500 | 0 | 0 | 0 | 5 | 9 | 12 |
| 100 | 3.000 | 2.985 | 15 | 0 | 0 | 5 | 7 | 10 |
| 150 | 4.501 | 3.000 | 1.501 | 0 | 0 | 5 | 9 | 16 |

Na oferta 100/s, 0,5% de throttling; sucesso realizado de 99,5/s. Em 150/s, 100 sucessos/s e 33,35% de rejeição: consistente com o teto do limitador, sem demonstrar saturação máxima de CPU/DB. A iteração adicional no último cenário é registrada, sem ajustar contagens para parecerem exatas. A taxa `http_req_failed` inclui os 429; não indica erros internos nesses casos.

Oito amostras de recursos por cenário, nenhuma falha de coleta. Picos amostrados de CPU API/DB, respectivamente: 8,41/8,02%; 14,29/14,30%; 20,52/20,65%; 24,20/25,96%. Máximo de conexões adquiridas observado no pool: 1 de 5. Memória amostrada máxima geral: API 55,26 MiB; DB 55,85 MiB. Esses números são amostras esparsas, não picos contínuos nem prova de ausência de filas. Cotas não equivalem a CPUs dedicadas.

## Verificação e limites

Ensaio real passou: 9.301 requests, 7.785 respostas 200 válidas, 1.516 respostas 429, zero outros erros, zero iterações descartadas, todas as coletas aprovadas. Python/JS passaram nas verificações de sintaxe. Docker remoto e sobrescrita de relatório existente foram recusados antes de provisionar recursos. Contagens de resultados, oferta/iterações e ausência de 429 em 10/50 foram conferidas no JSON. Containers e volumes do projeto sintético foram removidos; stack de desenvolvimento permaneceu saudável. Não foi repetida a suíte de aplicação/browser: não houve alteração de API, UI, SDK, dependências ou schema (5).

p95 de sucesso ficou abaixo de 100 ms nos quatro cenários desta fixture. Isso não demonstra SLO sustentado de produção: apenas uma flag habilitada a 100%, um token, 30 s por cenário e rede local. Batch, rollout parcial, dados maiores, concorrência de escrita, soak, falhas, TLS/proxy e capacidade da VPS continuam sem medição. A carga permanece manual; não foi adicionado gate de latência instável ao CI compartilhado. Não foi executado CI remoto.

Reprodução/condições em [runbook](runbooks/capacity.md), decisão em [ADR-014](adr/014-local-capacity.md). Próximo incremento proposto: preparar artefatos revisáveis de release (scans de imagens/SBOM, digests e ensaio local de rollback), mantendo publicação, licença e implantação externa pendentes. Coletor/alertas e backup recorrente real continuam pendentes; este benchmark não os configura.
