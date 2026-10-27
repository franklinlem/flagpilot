# Homologação externa — incremento 7A

25/10/2026. **Homologação disponível em https://flagpilot.franklem.uk**, gerenciada pelo Coolify de Contabo. Implantação e GitHub autorizados expressamente por Franklin; backups no `frank-note2` e monitoramento no Uptime Kuma também autorizados. Produção continua fora desta etapa.

## Instalação

- Repositório privado [franklinlem/flagpilot](https://github.com/franklinlem/flagpilot); histórico enviado e verificado com Gitleaks. Licença pública ainda pendente. Sem CD automático, publicação de imagens/npm ou release pública.
- Projeto Coolify `FlagPilot`, ambiente `homologacao`, serviço `flagpilot-homologacao` (`lsqrajvwfjp3rxb4xsxupnb1`). Recursos exclusivos; 30 containers preexistentes na VPS mantiveram seus IDs. Nenhum serviço de terceiros reiniciado.
- Quatro imagens construídas e analisadas no notebook, transportadas por SSH; image IDs conferidos na VPS. [Inventário](evidence/images-homolog-2026-10-25.json): zero HIGH/CRITICAL; dois UNKNOWN da API seguem a triagem OpenPGP de 6F, sem supressão. Sem build na VPS ou pull de tags mutáveis.
- Traefik existente termina HTTPS; certificado público validado. Somente web conectado à rede externa `coolify`. DB/API sem portas publicadas; redes internas adicionais, limites de CPU/memória e rotação de logs. Migrations concluíram com exit 0; DB/API/web saudáveis. O Coolify também conecta os componentes a uma bridge exclusiva do serviço, não marcada internal; portanto não se afirma bloqueio total de egress ou segmentação estrita entre esses quatro componentes. Nenhum DB/API entra na rede compartilhada de ingresso.
- PostgreSQL 16.15, schema 5, TLS interno obrigatório com `verify-full`. Conta runtime e migrator sem superuser/createdb/createrole; aplicação sem DDL. `APP_ENV=production` ativa as proteções técnicas de HTTPS/cookies, embora o ambiente operacional seja homologação.
- Banco novo independente do notebook. Administrador `franklinlem@gmail.com`, organização `Franklin`. Senha inicial aleatória em `.tools/credentials/flagpilot-homolog-admin.txt`, permissão 0600, fora do Git. Nenhuma senha versionada em documentação. A senha inicial apareceu no contexto de uma mensagem posterior; foi recomendada sua substituição, sem presumir que já tenha sido feita. Credencial temporária do provisionamento revogada e túnel administrativo encerrado; API HTTP global do Coolify permanece desabilitada.

## Validação observada

HTTPS válido; login/cookie Secure, HttpOnly e SameSite=Lax; projeto com três ambientes; flag habilitada; avaliação com token de servidor; token sem acesso administrativo; revogação rejeita avaliação; logout invalida sessão; `/metrics` e readiness internos bloqueados publicamente. Avaliação é server-side: requisições com Origin de navegador são recusadas por desenho. Projeto demonstrativo `Validação da homologação` / flag `homolog_ok` mantidos; os tokens de validação foram revogados.

O primeiro início encontrou certificados/configuração pública sem permissão de leitura pelo UID do DB após transporte com umask 077. Permissões públicas corrigidas, chave privada mantida restrita; banco/roles iniciais concluídos sem excluir volume. Não havia dados de aplicação nessa inicialização. Runbook registra os modos necessários.

Seis fluxos Chromium e seis Firefox passaram localmente após corrigir duas corridas no harness de navegação e a preservação da porta efêmera. CI remota de código `481f112` aprovada: [execução 40214673901](https://github.com/franklinlem/flagpilot/actions/runs/40214673901). A CI de `65db966` também foi aprovada: [execução 40215731673](https://github.com/franklinlem/flagpilot/actions/runs/40215731673). Franklin confirmou acesso, login e navegação no domínio em 25/10/2026; essa aceitação cobre os cenários relatados.

## Backup e monitoramento

Backup completo diário às **03h15, fuso -03**, puxado pelo `frank-note2`, retenção de 14 dias. Diretório `/home/franklin/backups/flagpilot/homolog` 0700. SSH dedicado com comando forçado e sem forwarding; pg_dump custom criptografado com GPG antes do transporte, somente chave pública na VPS. Chave privada de recuperação no notebook: `.tools/credentials/flagpilot-backup-private.asc` (0600); deve ser guardada também em cofre seguro independente. Sem essa chave não há restauração.

Transferência real executada e backup descriptografado diretamente para banco temporário local sem rede/portas, PGDATA em tmpfs; conteúdo das 11 tabelas comparado integralmente em memória e idêntico. Container removido, nenhum dump em claro persistido. [Evidência agregada](evidence/backup-homolog-2026-10-25.json). Cron instalado e serviço cron ativo; execução futura agendada ainda depende de o `frank-note2` estar ligado/conectado.

Uptime Kuma existente no `frank-note2`: monitores 14 (HTTPS do painel) e 15 (API `/api/v1/health/live` com conteúdo `"status":"ok"`), intervalo 60 s, timeout 20 s, duas tentativas com intervalo 30 s, validação TLS ativa. Primeiros heartbeats UP. Os 13 monitores anteriores preservados. Canal existente `backup hermes` vinculado por autorização de Franklin; sem enviar notificação artificial. Entrega real de alerta ainda não ensaiada.

## Limites

Monitor externo não substitui coletor de métricas nem verifica integridade do backup. Falha do cron fica no log/last-success; alerta automático de atraso de backup permanece pendente. Não há SLO/RPO/RTO de produção demonstrado. Retenção só no host físico, sem segunda cópia remota automática. Certificado TLS interno exige renovação antes de um ano; chave GPG antes de dois anos. Produção, licença, assinatura/provenance e publicação por digest de registry continuam pendentes.

Operação: [runbook de homologação](runbooks/homologation.md), [backups](runbooks/backups.md), [ADR-017](adr/017-isolated-homologation.md).

## Atualização de 27/10/2026 — 7B

Encaminhamento do Traefik fixado explicitamente na rede estável `coolify`, removendo a dependência da reconexão manual à rede exclusiva após recriar o proxy. Correção persistida no Coolify e no Compose, somente web reaplicado; regressão real da conexão perdida aprovada. Backups agendados de 26 e 27/10 observados no host físico. Causa, testes e limites em [INGRESS_STATUS.md](INGRESS_STATUS.md).

## Atualização de 27/10/2026 — 8A

Troca de senha pessoal disponível em Conta → Segurança da conta. Confirma senha atual, encerra todas as sessões e audita a mudança nas organizações atuais. API/web atualizados com DB/schema/proxy preservados e ingresso estável. Nova medição detectou curl/libcurl afetados no web; pacotes corrigidos e gate final aprovado antes de aplicar. Backups/monitores conferidos. Estado, resultados e limites em [ACCOUNT_SECURITY_STATUS.md](ACCOUNT_SECURITY_STATUS.md); próximas frentes em [ROADMAP.md](ROADMAP.md). Nenhuma senha real alterada automaticamente.
