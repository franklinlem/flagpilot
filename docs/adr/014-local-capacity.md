# ADR-014 — Benchmark local com limites ativos

Data: 25/10/2026. Estado: implementada localmente em 6D.

Atender aos cenários k6 10/50/100 req/s da especificação sem adicionar dependências ao runtime. Usar imagem oficial k6 1.6.1 fixada por digest e executor aberto com chegada constante; não usar loops sequenciais que reduzam a oferta quando o serviço fica lento. Cotas explícitas e recursos sintéticos em projeto Compose próprio tornam o experimento reproduzível sem operar sobre dados reais. Rede host do gerador restrita ao harness Linux/local, endpoint gerado internamente e loopback aleatório; não há opção de apontar carga a um serviço externo.

Manter limitador por token/peer/global e pool originais. Em 100/s a janela/jitter pode causar 429: separar rejeições de erros e de sucessos corretos, com latências dos sucessos e de todos os resultados. Adicionar cenário 150/s para observar throttling controlado, sem prometer throughput acima dos limites. Descartes do gerador e erros de validação invalidam a execução; p95 de máquina compartilhada fica registrado, sem gate rígido de performance em CI. Carga completa é comando manual local, não nova etapa recorrente no CI.

Relatório agregado contém condições, imagem/revisão, percentis, contadores e amostras de recursos; não contém URL concreta, token, senha, cookie, corpo ou contexto. Todos os serviços/volumes/arquivos temporários próprios são removidos no fim. Não alterar migrations, frontend, API, políticas de segurança ou infraestrutura externa neste incremento.

O experimento pequeno/curto orienta trabalho posterior; não mede máximo de saturação nem certifica SLO/RPO/RTO. Batch, rollout parcial, dataset maior, soak, concorrência de escrita e referência da VPS continuam pendentes. Método e fontes oficiais em [runbook](../runbooks/capacity.md).
