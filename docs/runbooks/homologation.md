# Homologação no Coolify

Instância atual, autorizações e limites em [HOMOLOGATION_STATUS.md](../HOMOLOGATION_STATUS.md). HTTPS: https://flagpilot.franklem.uk. Conta inicial em arquivo privado do notebook, nunca no repositório. Banco independente do desenvolvimento.

## Receita para atualização manual

1. Aprovar a revisão de código e concluir CI/scans. `make release-artifacts ARGS='--output .tools/releases/NOVA-REVISAO --report docs/evidence/NOVO-RELATORIO.json --keep-images'` preserva as tags locais analisadas; o padrão sem a opção continua limpando-as. Só prosseguir com gate aprovado e triagem dos demais avisos. Conferir manifest/image IDs, não inferir identidade pela tag.
2. Obter backup criptografado recente e verificar restauração antes de mudanças no banco. Não usar `down -v` ou migration down no banco existente. Confirmar compatibilidade de schema/reversão conforme [release](release.md).
3. Transportar somente as imagens aprovadas via `docker save`/compressão/SSH/`docker load`, conferindo `.Id` com o manifest. Não expor socket Docker pela rede. Manter imagens anteriores necessárias ao rollback e espaço disponível.
4. Usar `deploy/compose.homolog.yml` como receita Raw Compose do serviço próprio. Preencher `deploy/homolog.env.example` fora do Git; senhas aleatórias hexadecimais de pelo menos 48 caracteres, distintas entre owner/app/migrator, nunca em argumentos/logs. No Coolify, variáveis secretas pertencem somente ao serviço FlagPilot. Resolver caminhos e referências das imagens para o host; manter variáveis das senhas no gerenciador. Nunca imprimir o Compose renderizado.
5. Instalar migrations/configuração em `/home/franklin/flagpilot/homolog/release` e certificados em `tls`. Diretórios públicos do release 0755/arquivos 0644; CA/certificado público 0644, chave do servidor 0600 pertencendo ao UID PostgreSQL da imagem (70 nesta revisão). CA privada não vai ao servidor. Verificar leitura efetiva pelos usuários dos containers antes do primeiro start; umask 077 durante extração exige corrigir modos dos arquivos públicos. Não relaxar a chave privada. SAN inclui DNS `db`, serverAuth; API/migrator validam CA/hostname e DB rejeita TCP sem TLS.
6. Projeto `FlagPilot`, ambiente `homologacao`, serviço `flagpilot-homologacao`, UUID `lsqrajvwfjp3rxb4xsxupnb1`. Apenas web na rede externa `coolify` (o Coolify acrescenta também uma bridge exclusiva do serviço aos componentes; as redes internas adicionais não garantem isolamento estrito/egress bloqueado), endereço Coolify `https://flagpilot.franklem.uk:8080` (porta interna; URL pública não usa porta). Excluir job `migrate` do status contínuo, mantendo dependência `service_completed_successfully`. Não modificar o proxy compartilhado ou habilitar API HTTP global para operar este projeto.
7. Iniciar manualmente o serviço no Coolify. Confirmar migration exit 0; DB/API/web saudáveis; nenhuma porta DB/API publicada; HTTPS válido e avaliação autenticada/revogação funcionando. Conferir logs restritos ao projeto, com redação de segredos. Conferir monitores Uptime Kuma e inventário dos demais serviços.

Bootstrap somente para instalação realmente vazia: executar `/admin bootstrap --email EMAIL --org NOME --slug SLUG` no container API por terminal interativo. Automação aceita `--password-stdin` por canal controlado, nunca flag/variável de senha administrativa; gravar credencial inicial em arquivo privado, não no chat. Recuperação: `/admin reset-password --email EMAIL`, que revoga sessões. Não criar usuários por SQL.

Scripts de init só rodam em PGDATA vazio. Falha antes de criar banco/roles não se resolve excluindo volume: diagnosticar permissões e estado, concluir somente passos faltantes após confirmar ausência de dados/roles. Após aplicação inicializada, usar migrations versionadas; não reaplicar scripts CREATE ROLE cegamente.

## Consultas de saúde sem segredos

```sh
curl --fail https://flagpilot.franklem.uk/
curl --fail https://flagpilot.franklem.uk/api/v1/health/live
ssh contabo 'docker ps -a --filter label=com.docker.compose.project=lsqrajvwfjp3rxb4xsxupnb1 --format "{{.Names}} {{.Status}}"'
```

A label `traefik.docker.network` do web deve coincidir com o nome real da rede de ingresso (`coolify` nesta instância), também declarada persistentemente pelo proxy. Coolify pode anexar o proxy à rede exclusiva do serviço em runtime; essa conexão pode desaparecer em uma recriação e não deve ser necessária para o FlagPilot. Conferir Raw Compose, configuração gerada e label do container após alterações. Diagnóstico/regressão em [INGRESS_STATUS.md](../INGRESS_STATUS.md). Não reiniciar o proxy compartilhado para testar um único projeto.

Não disponibilizar readiness/métricas publicamente. O live confirma processo; teste funcional e healthchecks internos cobrem DB/migrations. Uptime Kuma monitores 14/15 no frank-note2, validação TLS, 60 s, timeout 20 s, duas tentativas. Canal autorizado `backup hermes`; não alterar credenciais/destinos do canal compartilhado. Entrega real de alerta não foi ensaiada nesta etapa.

Renovar certificado DB antes de 25/10/2027, preservando CA e SAN, chave restrita e reinício controlado somente deste projeto; testar verify-full e healthchecks. Conferir data real com openssl. Manter CA privada e chave de recuperação em cofre independente. Não promover esta instalação a produção automaticamente.
