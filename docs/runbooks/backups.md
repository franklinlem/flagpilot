# Backups da homologação

Origem Contabo; destino físico `ssh frank-note2`. Diretório exclusivo `/home/franklin/backups/flagpilot/homolog` (0700). Cron diário 03h15 (-03), retenção de 14 dias, dump PostgreSQL custom criptografado GPG. Chave privada de recuperação está no notebook, `.tools/credentials/flagpilot-backup-private.asc` (0600); guardar cópia em cofre seguro. Apenas destinatário público na VPS. Certificado CA privado do DB é material diferente, também deve ser preservado fora da VPS.

`deploy/backup/export.sh` é instalado na VPS como `/home/franklin/flagpilot/homolog/private/export-backup.sh` (0700), com `backup.env` (0600) contendo `FLAGPILOT_SERVICE_UUID` e fingerprint `FLAGPILOT_GPG_RECIPIENT`. Keyring público em `private/gnupg` (0700). Seleção estrita por labels do DB; pg_dump por socket local, criptografia antes do transporte; não gera dump em claro no disco. Lock impede sobreposição.

`deploy/backup/pull.sh` é instalado no destino como `pull.sh` (0700), junto de `pull_ed25519` (0600) e `known_hosts` com hostkey conferida pelo SSH já confiável. Na VPS, somente essa chave recebe opção `command="/home/franklin/flagpilot/homolog/private/export-backup.sh",restrict`. Preservar outras entradas de authorized_keys. A chave do destino não permite shell, forwarding, restore ou exclusão de dados na origem. VPS não precisa alcançar Tailscale do host físico.

Cron existente preservado; bloco exclusivo:

```cron
# BEGIN FlagPilot homolog backup
15 3 * * * /home/franklin/backups/flagpilot/homolog/pull.sh >> /home/franklin/backups/flagpilot/homolog/pull.log 2>&1
# END FlagPilot homolog backup
```

## Conferir e executar

```sh
ssh frank-note2 '/home/franklin/backups/flagpilot/homolog/pull.sh'
ssh frank-note2 'cat /home/franklin/backups/flagpilot/homolog/last-success.txt; systemctl is-active cron'
```

Cada arquivo tem `.sha256` e timestamp UTC. Confira checksum no destino antes de copiar/restaurar; ele detecta corrupção acidental, não autentica origem contra invasor do próprio host. Transferência SSH autentica o host; GPG protege confidencialidade/integridade do conteúdo. A rotina conserva arquivo anterior em falhas, não avança last-success e escreve erro em pull.log. A limpeza aplica-se somente aos nomes próprios no diretório próprio. Conferir espaço e tamanho do log periodicamente; rotação automática de pull.log ainda não configurada. Host físico desligado/rede indisponível impede a execução; alerta automático de backup atrasado ainda pendente. Monitores HTTPS/API não provam backup recente.

## Restaurar

1. Copiar `.dump.gpg` e checksum via SSH para diretório privado em destino isolado. Conferir SHA-256. Importar a chave privada de recuperação em GNUPGHOME temporário 0700 protegido; não importar chave em hosts compartilhados sem necessidade, nem imprimir material secreto.
2. Preparar PostgreSQL 16 compatível, banco **vazio**, roles `flagpilot_app` e `flagpilot_migrator` antes dos grants; usar credenciais novas. Não restaurar sobre o banco ativo. Na validação 7A o destino era container local, `--network none`, sem portas, PGDATA em tmpfs e limite de recursos. Garantir espaço suficiente para bases maiores.
3. Canalizar `gpg --batch --decrypt ARQUIVO.dump.gpg` diretamente para `pg_restore --no-owner --exit-on-error --single-transaction -U OWNER -d flagpilot`, por stdin do container. Usar pipefail e conferir saída dos dois processos. Nunca gravar plaintext ou imprimir dados no terminal. O banco pode conter hashes de senha/sessão e dados pessoais; proteger também o destino.
4. Conferir schema, roles/grants, todas as tabelas e conteúdo, além de fluxo funcional isolado. Para recuperação de incidente, revogar sessões/tokens restaurados e rotacionar credenciais antes de servir tráfego, conforme [recuperação](recovery.md). Restaurar backup também restaura o estado de revogação daquela data.
5. Destruir somente o destino descartável ao terminar. Preservar os backups criptografados e cofre de chaves.

[Evidência real de 25/10/2026](../evidence/backup-homolog-2026-10-25.json): descriptografia/restore aprovados, 11 tabelas idênticas à origem comparadas em memória, nenhum dump em claro persistido e destino removido. Não é prova de RPO/RTO produtivos. GPG expira em dois anos; renovar destinatário antes do vencimento e manter chaves antigas enquanto backups correspondentes existirem. Sem segunda cópia externa automática.
