# Identidade administrativa — operação local

Incremento 2A, Linux/Debian. Sem signup público, conta padrão ou senha pré-configurada. Operador com acesso local às credenciais do banco é uma fronteira privilegiada; esses comandos não devem ser expostos em HTTP.

## Primeiro administrador

Com a stack local atualizada (`make dev-containers`) e migrations aplicadas:

```sh
make bootstrap
```

O comando compila o binário administrativo local, carrega `.env` sem executar shell e solicita email, nome/slug da organização e senha confirmada sem echo. Requer Go (pode estar em `.tools/go`) e acesso ao DB publicado em loopback. Exemplo com metadados antecipados, mantendo senha oculta:

```sh
make bootstrap ARGS='--email seu-email@example.com --org MinhaEquipe --slug minha-equipe'
```

Alternativa sem Go no host, usando o binário da imagem em container:

```sh
docker compose --env-file .env -f deploy/compose.dev.yml exec api /admin bootstrap
```

Bootstrap só funciona em instalação vazia: cria usuário, organização, associação Owner e auditoria em uma única transação. Lock transacional impede duas inicializações concorrentes. Instalação já inicializada retorna erro sem alterar dados. Não inserir usuário por SQL para contornar essa condição.

Senha: pelo menos 12 caracteres, no máximo 256 bytes UTF-8, sem truncamento silencioso. Espaços nas extremidades fazem parte da senha; senha apenas de espaços é rejeitada. Armazenada com Argon2id (19 MiB, 2 iterações, paralelismo 1, salt aleatório 16 bytes e hash 32 bytes), sem segredo global adicional obrigatório. O formato é versionado; alteração futura de custo deve acompanhar política de rehash.

Depois, entrar em http://127.0.0.1:5173 usando a conta criada. Não compartilhar senha no chat ou salvar em manifests. Nenhuma conta permanente de Franklin foi criada automaticamente.

## Recuperar senha

```sh
make reset-password ARGS='--email seu-email@example.com'
```

Ou `docker compose ... exec api /admin reset-password --email seu-email@example.com`. Solicita e confirma senha oculta; atualiza hash, revoga **todas** as sessões e insere auditoria na mesma transação. Se a auditoria falhar, a senha e sessões anteriores permanecem. Não reativa conta desativada; gestão de membros/desativação pelo produto será tratada no próximo incremento.

Para automação controlada, `--password-stdin` lê uma linha de stdin e exige metadados como flags. Nunca existe flag `--password`, senha em argumentos ou variável de senha administrativa no `.env`. O E2E usa senha aleatória em memória e stdin; não reutilizar credenciais sintéticas fora dos testes. No terminal interativo, Ctrl+C restaura o echo e encerra com código 130; não há traceback ou alteração antes da transação.

## Sessões e browser

- Cookie host-only `flagpilot_session`, HttpOnly, SameSite=Lax, Path=/; Secure em produção/HTTPS. HTTP só é permitido nas origens loopback de desenvolvimento. Login gera novo token e revoga o cookie anterior quando existente.
- Token aleatório 256 bits em hexadecimal; somente SHA-256 no banco. Expiração absoluta por `SESSION_TTL`, padrão 12h, limites 1h–168h. Logout, expiração, recuperação e desativação do usuário invalidam acesso no servidor.
- CSRF é HMAC-SHA-256 com chave da própria sessão e domínio `flagpilot/csrf/v1`, devolvido em login/me e guardado apenas em memória no frontend. Não há necessidade de SESSION_SECRET/CSRF_SECRET globais para este desenho. Mutações exigem `X-CSRF-Token` vinculado à sessão e `Origin` exata permitida.
- Login exige JSON e Origin permitida mesmo sem sessão prévia. Sec-Fetch-Site cross-site é rejeitado. Sem CORS permissivo. Cookies e CSRF não ficam em localStorage/sessionStorage.
- `PUBLIC_BASE_URL` define origem do painel; `ALLOWED_ORIGINS` aceita lista explícita separada por vírgula, sem path/query/wildcards. Em modo host, configure os valores em `.env` se mudar a porta. No Compose há defaults para 127.0.0.1 e localhost:5173. Produção exige origem HTTPS explícita.
- Limites de login: 10 tentativas por IP e por conta em 15 minutos, global 120/min e duas verificações Argon2 concorrentes por processo. Mapa limitado a 4096 chaves com expiração; respostas 429 incluem Retry-After. Limites são locais a uma réplica, sem promessa distribuída.
- Não confiar em X-Forwarded-For externo. No proxy nginx o limite por IP pode agregar clientes; confiar em hops específicos precisa ser definido/validado antes da homologação, sem abrir bypass por header arbitrário.

## Auditoria e limites

Migration 000002 cria audit_events com grants SELECT/INSERT ao runtime, sem UPDATE/DELETE. Eventos `installation.bootstrap`, `auth.login`, `auth.logout`, `auth.password_reset` usam ator, organização e alvo, sem senha, hash de senha, token, CSRF ou contexto pessoal. Não há garantia de imutabilidade contra administrador do banco. Falhas de login retornam resposta genérica e logs HTTP sem email ou corpo; não há persistência de tentativa por email.

Ainda não há gestão de membros, convites, RBAC de recursos, projetos, flags ou auditoria consultável por UI. A tela autenticada lista somente organizações do próprio usuário via `/me`; isso não representa a entrega dos endpoints de domínio. Retenção/limpeza de sessões expiradas e auditoria será um job controlado futuro, sem purga automática nesta etapa.

Referências do desenho: [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [OWASP CSRF](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html), [Go Argon2](https://pkg.go.dev/golang.org/x/crypto/argon2).
