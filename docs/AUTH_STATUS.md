# Incremento 2A — identidade administrativa

Data: 24/10/2026. Incremento entregue e validado localmente; GitHub Actions configurado, sem execução remota nesta sessão.

## Entrega

- Bootstrap CLI transacional/auditável, senha oculta e confirmação; segunda inicialização negada e corrida de bootstrap serializada.
- Recuperação local de senha com revogação de todas as sessões e auditoria atômica.
- POST `/api/v1/auth/login`, POST `/api/v1/auth/logout` e GET `/api/v1/me`; contrato, respostas e tipos OpenAPI atualizados.
- Argon2id, cookie opaco HttpOnly/SameSite, rotação, expiração absoluta, invalidação de conta desativada, CSRF vinculado à sessão, Origin estrita e limites de login/JSON/concorrência.
- Migration 000002_audit e readiness compatível com schema 2. Audit SELECT/INSERT para runtime; UPDATE/DELETE negados.
- Tela de login com validação e mensagens de erro, página autenticada da conta/organizações e logout; credenciais/tokens fora dos storages do browser.
- CI com testes de terminal e E2E Chromium em stack descartável, sem publicar/deploy.

## Evidências locais

- `make ci`: lint Go/TS, Spectral, tipos, unitários/race, integração PostgreSQL e builds.
- Integração: upgrade schema 1→2, down/up somente descartável, unicidade/FKs, bootstrap concorrente, falha de auditoria com rollback, sessão como hash, rotação, CSRF, origem, cookie Secure, expiração, desativação, recuperação e rate limit.
- Vitest/Testing Library: formulário, credenciais inválidas, login/logout, falha de rede/retentativa, sessão restaurada e logout indisponível.
- `python3 scripts/check-admin-terminal.py`: senha oculta, confirmação, restauração de termios e SIGINT com exit 130, sem acesso ao DB.
- `python3 scripts/auth-e2e.py`: build API + nginx estático, bootstrap CLI com credencial aleatória via stdin, browser Chromium real, senha incorreta, login, perfil/organização/Owner, reload por cookie, storage vazio, CSRF e cookie revogado após logout. Stack/volume efêmeros removidos.
- `govulncheck` e `npm audit`: sem vulnerabilidade alcançável em código/pacotes usados e zero achados npm. Crypto atualizado para 0.58.0; a base Go ainda sinaliza [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) no pacote **openpgp não importado** desse módulo, sem correção disponível. O projeto não utiliza OpenPGP; esse pacote não deve entrar no grafo de imports. Não declarar zero achados em todo o módulo.
- Smoke de resiliência: live/ready, SIGINT com exit 0, reinício e queda real do DB. Probes PostgreSQL agora usam TCP, evitando considerar pronto o servidor temporário de inicialização da imagem oficial. Logs de diagnóstico são sanitizados antes da limpeza da stack.
- Gitleaks e verificação de arquivos gerados antes do commit.

## Uso e limites

Atualizar a página em http://127.0.0.1:5173 para a tela de login. Criar sua conta por `make bootstrap`; não há senha padrão nem conta permanente criada pelo assistente. [Runbook de identidade](runbooks/identity.md).

A captura inicial do usuário confirmou a página da fundação com “Serviço disponível”; não havia falha de conectividade na captura. Franklin confirmou manualmente a criação da senha e o login e anexou captura com conta autenticada, organização franklem e papel Proprietário. A validação cobre somente esses cenários. E2E headless não equivale a aceitação manual de acessibilidade ou dashboard completo.

Estado histórico de 2A; organizações/membros e sua autorização foram entregues em [2B](ORGANIZATIONS_STATUS.md). Ainda não entregues: RBAC dos demais recursos, projetos/ambientes, CRUD de flags, avaliação, SDK funcional, métricas/monitoramento de produção e recuperação operacional. Sem deploy, DNS, publicação de imagens/pacotes ou alteração em servidores externos. Próximo incremento: 2B com autorização/isolamento, invariantes do último Owner e mutações auditadas.
