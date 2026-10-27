# ADR-017 — Homologação isolada e backup pull criptografado

Data: 25/10/2026. Estado: implementada em 7A, por autorização explícita.

Usar projeto/ambiente/serviço próprios no Coolify existente, banco novo e domínio flagpilot.franklem.uk. Transportar as quatro imagens locais previamente escaneadas por SSH, comparar image IDs, sem registry ou deploy automático. Repositório privado enquanto licença/publicação não forem decididas. Coolify continua com API HTTP desabilitada; provisionamento administrativo pontual pelo SSH existente e controladores da própria versão instalada, sem editar tabelas de configuração diretamente. Revogar credencial temporária ao terminar.

Somente web entra na rede de ingresso Traefik; redes internas separam API/DB, sem portas publicadas. DB usa CA própria, `verify-full` e rejeita TCP sem TLS; migrator separado do runtime e ambos sem privilégios de superuser. Limitar recursos para compartilhar a VPS sem reiniciar outros projetos. Execução de migration one-shot não integra o indicador contínuo de saúde, mas API depende de sua conclusão bem-sucedida.

Backups pertencem ao host físico frank-note2. A VPS não alcança seu endereço Tailscale: adotar pull pela porta SSH pública existente da VPS, com chave dedicada, hostkey fixada e comando forçado exclusivo de exportação. Não instalar VPN nem conceder shell com essa chave. Criptografar pg_dump na origem com destinatário GPG; chave privada de recuperação permanece fora dos dois hosts. Rotina cron diária preserva jobs anteriores e aplica retenção somente no diretório/prefixo próprios. Testar restauração real em ambiente descartável antes de declarar rotina configurada.

Reutilizar Uptime Kuma no host físico para HTTPS/API, sem mudar outros monitores; notificações vinculadas somente após escolha explícita do canal. Monitoramento de atraso do backup e prova de entrega de alerta são trabalhos separados. Ver resultados e limites em [HOMOLOGATION_STATUS.md](../HOMOLOGATION_STATUS.md).

Complemento 7B (27/10/2026): definir explicitamente a rede do router Traefik no web para coincidir com o ingresso estável. A rede extra do serviço anexada em runtime ao proxy se perde em sua recriação; não depender dela. Preferir esta correção restrita ao FlagPilot a modificar a rotina global de atualização do proxy compartilhado. Remover a conexão extra do próprio serviço reproduz a condição da falha sem recriar o proxy; limites/evidência em [INGRESS_STATUS.md](../INGRESS_STATUS.md).
