# ADR-011 — Tokens e avaliação booleana

Data: 24/10/2026. Estado: implementada na etapa 4A, após continuidade autorizada por Franklin.

## Credenciais e administração

- Módulos `tokens` e `evaluation`, sem dependências novas. Owner/Admin emitem, listam metadados e revogam no escopo organizacional; Editor/Viewer não administram tokens. A associação ativa é relida sob SHARE na organização, reutilizando `projects.Authorize`.
- Token `fp_eval_` + 32 bytes aleatórios codificados em hexadecimal. Banco guarda somente SHA-256 do token completo, com índice único. Prefixo de identificação contém os primeiros 8 caracteres aleatórios e não serve como credencial. Os demais 224 bits continuam secretos mesmo com o prefixo divulgado. Não há recuperação do segredo nem rotação que reutilize o token.
- Escopo único `evaluation`, ambiente/projeto imutáveis e FK composta; permissões SQL permitem SELECT/INSERT e UPDATE somente de revoked_at. `created_by` registra origem administrativa; a credencial pertence ao ambiente, não à sessão pessoal do emissor. Remover o emissor da organização ou desativar sua conta não substitui a revogação explícita da credencial de serviço.
- Validade futura obrigatória, até 365 dias; UI sugere 90 dias e oferece 1/7/30/90/365. Datas UTC no contrato; sem token permanente. A emissão revalida o prazo após adquirir autorização. Não adicionar segredo de avaliação às variáveis do próprio servidor FlagPilot: ele é configurado no backend consumidor.
- Retorno secreto somente no POST de emissão, após commit. GET/auditoria contêm metadados e prefixo, sem segredo/hash. UI usa somente estado local do componente, sem Query/Mutation cache, storage do navegador ou URL. Ocultar/fechar/sair descarta a exibição; clipboard só é usado por clique explícito. Não há promessa de apagamento criptográfico da memória nem de remoção do clipboard gerenciado pelo usuário.
- Revogação idempotente (204), com um único evento para a alteração real. Emissão/revogação e auditoria são atômicas. Se a resposta de emissão se perder, não repetir automaticamente: localizar a credencial pelos metadados, revogá-la e emitir outra.

## Autenticação e snapshot

Avaliação aceita somente Authorization Bearer de formato explícito, sem fallback para sessão/cookie. Bearer não administra a aplicação. Requisições com Origin ou Sec-Fetch-Site são recusadas: chave secreta é para servidor/BFF; não oferecer CORS nem chave pública nesta etapa. BFF futuro deverá autorizar usuários/flags, sem proxy aberto.

Cada avaliação abre transação e verifica hash, escopo, revogação e expiração no banco, sem cache positivo. SHARE no token impede a revogação de concluir enquanto uma avaliação autorizada está em voo; expiração é relida após obter o lock. Depois adquire SHARE no ambiente e lê todas as configurações solicitadas numa consulta. Não bloqueia organização/projeto. Revogação usa organização → token, sem bloquear ambiente, evitando ciclo de locks. Uma chamada já autorizada pode concluir; chamadas posteriores à revogação/expiração são rejeitadas.

O ambiente é inferido exclusivamente da credencial. Campos de projeto/ambiente enviados no corpo são desconhecidos e rejeitados. Batch preserva ordem e duplicatas, com uma única revisão para todos os resultados e para o envelope.

## Semântica e limites

- Ausente/arquivada: false/NOT_FOUND. Desativada ou rollout 0: false/DISABLED. Habilitada e rollout 10000: true/ENABLED.
- Rollout parcial continua fora de 4A: não é aceito pela API administrativa atual. Se um operador SQL privilegiado introduzir uma configuração parcial, avaliação retorna 503, permitindo fallback fechado no consumidor, sem inventar um resultado ENABLED. Percentuais/bucketing/MISSING_CONTEXT serão implementados em 4B.
- Contexto opcional apenas em memória; bucketingKey não vazio, até 256 bytes UTF-8. Não vai ao banco, auditoria, métricas ou logs; sem segmentação nesta etapa.
- Body JSON estrito até 64 KiB; batch 1–100, incluindo duplicatas. Sem retorno de salt, nomes de usuários, hashes ou IDs administrativos na avaliação.
- Rate limit fixo em memória por segundo: global 600, peer real 120, hash de credencial 100; mapa até 4096 entradas, expiração limitada, Retry-After 1 em 429. Aplica antes da consulta de credencial. Não confiar em X-Forwarded-For. Atrás de nginx/Traefik, clientes que compartilhem peer compartilham esse limite. Ajuste de proxy confiável/capacidade exige medição futura; limites não são promessa de throughput.

## Persistência e evidências

Migration aditiva 000005, readiness schema 5. Preserva identidades, projetos, ambientes, flags/configurações anteriores. Testes PostgreSQL cobrem upgrade com dados, hash/segredo, RBAC/IDOR, expiração, revogação, rollback e lote concorrente consistente; unidade cobre decisão e rate limit. Chromium cobre emissão única, storage vazio, avaliação por cliente de servidor, ambientes separados, revogação e Viewer sem gestão. Harness redige tokens de avaliação em saída/artefatos textuais de erro.

Sem SDK funcional, rollout parcial, cache de avaliação, Redis, métricas de uso ou alteração externa. Sem benchmark/SLO verificado. Próximas etapas: 4B (rollout) e 5 (SDK).
