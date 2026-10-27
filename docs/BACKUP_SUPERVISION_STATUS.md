# Backup supervisionado — incremento 8B

27/10/2026. Instalado no frank-note2, somente no diretório do FlagPilot. Nenhum serviço da aplicação, proxy ou banco foi recriado.

- Pull diário continua às 03h15 (-03), criptografia GPG antes do transporte e retenção de 14 dias. Novo pull real concluído em 27/10 às 16:21:57 UTC; arquivo e SHA-256 conferidos pela supervisão.
- `monitor.py` mantém metadados JSON atômicos 0600. Falha do pull conserva o último sucesso e registra tentativa falha. Tentativa acima de 30 minutos, relógio futuro acima de 5 minutos, arquivo/metadata ausente ou inválido, corrupção e sucesso com mais de 26 horas resultam DOWN.
- Cron independente a cada cinco minutos verifica metadados, arquivo e SHA-256. Push local ao Kuma usa segredo em `monitor.json` 0600; não imprime URL/token/erros de provedor. Redirects/endereços remotos são recusados. Verificação do hash tem orçamento de 15 segundos entre leituras; leitura de disco bloqueada depende também do SO.
- Monitor push **17**, intervalo de 600 s, sem retries, canal existente **backup hermes**. Ausência de push detecta parada do watcher/cron enquanto Kuma estiver funcionando. Monitoramento HTTPS/API 14/15 preservado e UP.
- `pull.log` passa por copytruncate quando alcançar 1 MiB, conservando dois históricos de até 1 MiB. Há pequena janela de perda de linhas durante copytruncate; não usar o log como registro contábil.

[Evidência](evidence/backup-supervision-2026-10-27.json): fixtures recentes/atrasadas/falhas/recuperadas foram enviadas ao monitor real; heartbeats confirmam UP → DOWN → UP e retorno ao estado verdadeiro. Arquivos e metadados reais preservados. O vínculo ao canal e a aceitação dos pushes foram verificados; leitura/recebimento pelo destinatário não foi confirmada.

Sete testes automatizados cobrem estados, relógio, corrupção, traversal, permissões, rotação e falha real do shell com SSH sintético. Sem novas dependências. [ADR-019](adr/019-backup-supervision.md), [runbook](runbooks/backups.md).

Limites: 26 h é limiar de alerta com tolerância operacional, não garantia de RPO de 24 h. Backup e Kuma compartilham o frank-note2: se o host inteiro desligar, ele não consegue emitir o próprio alerta. Segunda cópia independente e monitor externo continuam pendentes. O restore real de referência continua sendo o de 25/10; este incremento não alega novo restore completo.
