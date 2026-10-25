# Modelo de ameaças inicial

Registro inicial da fundação; desde 2A, autenticação e auditoria estão implementadas conforme [runbook de identidade](../runbooks/identity.md). A tabela abaixo preserva o levantamento inicial; acompanhar entrega e evidências em [AUTH_STATUS.md](../AUTH_STATUS.md). Direções aprovadas pelo plano de 24/10/2026; controles futuros abaixo não devem ser reportados como já implementados.

Ativos: senhas/hashes, sessões, tokens de avaliação, configuração por ambiente, auditoria, disponibilidade, dados de organizações e credenciais de banco. Fronteiras: browser não confiável → proxy/painel → API; cliente servidor → API de avaliação; API → PostgreSQL; operador → migration/CLI; CI → dependências/imagens.

| Ameaça | Controle atual | Controle a entregar |
|---|---|---|
| Vazamento de credenciais | .env 0600/ignorado, template sem senha, logs por rota sem body/query/header | Redaction auditável em login/auditoria, secrets do destino e rotação |
| Banco comprometido por API | Runtime sem DDL, consultas pgx/sqlc, banco publicado só em loopback local | Menor privilégio por novas tabelas; papel de auditoria sem UPDATE/DELETE |
| Exposição da saúde interna | Ready sem detalhes; nginx runtime bloqueia a rota | Regras Traefik e API sem porta pública, a validar no destino |
| Falha de DB/schema | Ready 503, live independente, timeout DB e schema guard | Resiliência/backup/restore no destino |
| IDOR/escalada | Endpoints de domínio ainda inexistentes | Papéis org herdados, escopo em consulta/escrita e testes cruzados |
| CSRF/sessão roubada | Login ainda inexistente | SameSite/HttpOnly/Secure, CSRF e Origin, Argon2id, limites e revogação |
| Token browser | Nenhum token exposto, SDK sem cliente | BFF consumidor com allowlist, segredo só no backend |
| Alteração perdida | Sem escritas de negócio | Revisão atômica, If-Match 428/412, auditoria na mesma transação |
| XSS | React escaping, CSP no nginx runtime | Formulários validados, a11y e testes de payload sem HTML arbitrário |
| Abuso/replay | Timeouts/limite de headers no HTTP; proxy limita body | Rate limit por login/token/IP confiável e limites de batch/contexto |
| Supply chain | Lockfiles, versões de ferramentas, checksums, actions por SHA, CI sem secrets de deploy | SBOM, scan de imagens e assinatura/proveniência antes de release |

Não registrar bucketingKey em logs/banco/traces, nem usar identificadores pessoais como labels. Contexto futuro do SDK fica em memória limitada por TTL; preferir pseudônimos definidos na aplicação consumidora. Cache implica janela de propagação; flags não são mecanismo de autorização.

Riscos atuais: imagens de desenvolvimento fixadas por tag, ainda sem digest; scans de imagem/SBOM ficam para release; ambiente local sem TLS; usuário inicializador PostgreSQL é superuser apenas na stack local. Readiness exato exige atualizar constante/sqlc junto das próximas migrations. UI placeholder não constitui aceitação visual do dashboard. Autorização deve ser testada quando existir, não inferida deste schema.

## Atualização de identidade (2A)

Controles entregues: Argon2id, sessão revogável/rotacionada, origem/JSON/Fetch Metadata no login, HMAC CSRF por sessão no logout, TTL absoluto, limite por conta/IP/global e de trabalho Argon2, bootstrap/recuperação CLI e auditoria atômica. Token de sessão somente como hash; cookie HttpOnly; Secure obrigatório em produção. Testes com DB real e browser verificam estes controles.

Riscos residuais: uma réplica para rate limit; clientes atrás de proxy podem compartilhar limite por IP até definir hops confiáveis; purge de sessões/auditoria ainda não automatizado; RBAC de recursos ainda inexistente. Scanner Go informa GO-2026-5932 somente no pacote OpenPGP não importado de x/crypto 0.58.0; não utilizar esse pacote e reavaliar antes de adicionar imports criptográficos. Não há exceção a vulnerabilidade alcançável.

## Atualização de organizações/membros (2B)

Autorização negada por padrão e validada no banco para cada operação. Locks por organização com releitura do papel após espera, último Owner ativo e auditoria na mesma transação. IDOR em leitura/escrita, privilégios de Admin/Editor/Viewer e concorrência testados com DB real; cache separado e limpo por identidade no frontend. Cursor é entrada não confiável, nunca autorização. Não há pesquisa global de usuários; provisionamento local com Viewer por padrão. Operador SQL privilegiado pode burlar invariantes de aplicação; desativação global pelo produto precisará respeitar os Owners ativos de cada organização. Demais recursos ainda não possuem endpoints implementados.

## Incremento 3B — flags

Escritas exigem sessão/CSRF/Origin e papel organizacional relido no banco. Configuração bloqueia somente seu ambiente após proteger a autorização no escopo; mutações do catálogo e criação de ambientes serializam no projeto. FKs compostas impedem cruzar projetos. If-Match exato é obrigatório: sem wildcard nem sobrescrita silenciosa. Produção usa classificação persistida e confirmação explícita; confirmação não concede autorização. Arquivamento preserva a chave e exclui configurações das leituras ativas; o futuro avaliador também precisa filtrar archived_at. Auditoria/revisões/dados são atômicos, e salt não sai nos DTOs nem eventos. Nenhum token de avaliação ou endpoint público de avaliação foi adicionado. Ver ADR-010 e FLAGS_STATUS.md para decisões e evidências.

## Incremento 4A — credenciais de avaliação

Segredo aleatório com hash SHA-256, escopo evaluation e FK composta por ambiente/projeto; nunca em auditoria/listagem/cache/browser storage. Somente Owner/Admin administram. Avaliação usa Bearer sem fallback para cookie e rejeita Origin/Sec-Fetch-Site; browser precisa de BFF autorizado. Autenticação sem cache positivo e SHARE no token/snapshot ambiental, limitando revogação às chamadas já em voo. JSON/body/lote/contexto e taxa têm limites; peer usa conexão real, sem confiança automática em cabeçalhos de proxy. Contexto não é persistido nem logado. UI descarta exibição do segredo ao ocultar/navegar; harness redige tokens em saída/artefatos textuais. Credencial de serviço deve ser revogada explicitamente na rotação/saída de operadores. Ver ADR-011 para limites e condições de concorrência.

## Incrementos 5/6 — estado local

SDK usa credencial somente no backend, mapas limitados por instância/contexto e janela de cache documentada; falhas não viram valores válidos nem ganham revisão remota. Histórico exige sessão/associação ativa, projeta somente campos escalares conhecidos e não oferece escrita/restauração. Em 6B, timeout do painel não pode ser desativado pela presença de sinal de cancelamento; interromper uma request no browser não garante rollback de mutação já confirmada. Chromium/Firefox exercitam recuperação e sessão em stacks separadas, mantendo os limites de login. Limites e lacunas atuais em MVP_READINESS.md; seções anteriores são registros históricos, não inventário atualizado de todas as funcionalidades.
