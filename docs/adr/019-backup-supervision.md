# ADR-019 — Supervisão do backup por push local

Data: 27/10/2026. Estado: implementada em homologação (8B).

Manter pull criptografado e cron existentes, acrescentando Python stdlib para metadados atômicos, checksum e push ao Kuma local. Separar checagem de execução: uma falha do pull não impede o watcher de reportar; uma falha de entrega não transforma backup salvo em falha do dump.

Falha da última tentativa prevalece sobre sucesso anterior ainda recente. Atraso superior a 26 h e execução acima de 30 min resultam DOWN. Watcher a cada 5 min e monitor de ausência de push de 10 min. URL secreta restrita a loopback, arquivo 0600, sem redirects ou payload de banco nos alertas. Dois históricos limitados do log próprio; demais cron jobs/monitores preservados.

Evita novo daemon/exporter ou Prometheus obrigatório. Consequência: compartilha indisponibilidade do host físico com o Kuma, não substitui monitor externo nem prova recuperabilidade. O checksum detecta corrupção acidental, não autentica conteúdo contra invasor do host. Restauração deve continuar sendo ensaiada separadamente.
