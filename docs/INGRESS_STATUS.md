# Encaminhamento persistente da homologação — 7B

27/10/2026. Correção aplicada ao serviço FlagPilot, sem reiniciar/recriar o Traefik compartilhado ou alterar sua rotina global de atualização.

## Incidente e causa

Na [conversa compartilhada por Franklin](https://chatgpt.com/share/6ae0c17f-25f8-83e9-974c-b2679724ca2f), o painel/API foram recuperados reconectando manualmente o proxy à rede exclusiva do serviço. O registro fornecido em `checagem flagpilot contabo.txt` mostra a rotina `traefik-safe-update` recriando o proxy às 04h34 de 27/10/2026 (-03). A rotina verificava saúde do container e `/ping`, sem conferir as aplicações ou restaurar conexões de rede adicionadas em runtime. A versão exibida continuava 3.7.16, mas os image IDs diferiam.

Inspeção atual confirmou DB/API/web saudáveis, com `traefik.docker.network=lsqrajvwfjp3rxb4xsxupnb1` no web. Proxy e web também estavam conectados à rede estável `coolify`; ainda assim a label obrigava Traefik a usar a rede exclusiva. Sua perda após recriação interrompia o encaminhamento, embora o processo Traefik estivesse saudável.

## Correção

`deploy/compose.homolog.yml` agora define `traefik.docker.network: ${FLAGPILOT_INGRESS_NETWORK:-coolify}` somente no serviço web. Na instância atual a label é `coolify`, coerente com a rede externa de ingresso já declarada. A definição Raw Compose no Coolify recebeu somente essa label; o parser instalado preserva a escolha explícita. Configuração gerada e container em execução foram conferidos. A declaração persistente de `coolify` também foi confirmada no Compose do proxy.

O Coolify rejeitou o comando de start porque o serviço já estava ativo; nenhuma alteração de runtime ocorreu nessa tentativa. Para aplicar somente a mudança necessária, seus arquivos foram gravados pelo método nativo `saveComposeConfigs`, seguido de Compose `up -d --no-deps --no-build --pull never web`, exclusivamente no diretório/projeto FlagPilot. API, DB, migration concluída e proxy mantiveram os containers existentes; não houve novo build, migration, alteração de dados ou deploy de produção. Reaplicações futuras pelo painel usam a definição já persistida.

## Verificação

- Compose local expandido: label do router coincide com o nome real da rede de ingresso; DB/API/migrator não recebem a label.
- Raw Compose persistente, Compose gerado e web ativo apontam para `coolify`.
- Removida a conexão manual redundante entre `coolify-proxy` e a rede exclusiva **somente do FlagPilot**, reproduzindo a condição que causava a falha. O proxy manteve suas outras redes e seu container.
- HTTPS do painel e API retornaram 200, inclusive pela consulta originada no `frank-note2`; `/metrics` e readiness continuaram 404 publicamente. Web saudável. IDs dos demais containers preservados.
- [Evidência agregada](evidence/ingress-network-2026-10-27.json). Não foi executada recriação completa do proxy compartilhado: o teste removeu precisamente a conexão cuja ausência causava a falha.

A CI de 7A (`65db966`, execução 40215731673) concluiu com sucesso. Franklin confirmou acesso/login e navegação na homologação em 25/10; essa aceitação cobre os cenários relatados. As mudanças 7B não alteram código de negócio, contratos, schema ou dependências; validação direcionada por Compose e regressão real do encaminhamento.

## Continuidade operacional

Backups automáticos reais observados em 26 e 27/10 às 03h15 (-03), arquivos criptografados no frank-note2. A restauração integral validada em 7A continua sendo a evidência de restore; os dois novos arquivos não foram restaurados nesta etapa. Canal de alerta `backup hermes` e monitores existentes preservados. Após a regressão, os monitores 14/15 registraram novos heartbeats UP, ambos ainda vinculados ao canal autorizado.

A correção remove a dependência do FlagPilot da conexão manual com sua rede exclusiva. Não corrige globalmente `traefik-safe-update` nem garante conectividade de outros projetos que dependam de redes anexadas manualmente. Essa rotina ainda deve futuramente preservar redes no update/rollback e verificar aplicações; nenhuma alteração global foi feita por esta correção localizada. A rede estável `coolify` precisa continuar declarada na configuração persistente do proxy.
