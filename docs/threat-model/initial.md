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
