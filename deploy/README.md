# Desenvolvimento e containers

`compose.dev.yml` usa o project name `flagpilot-dev`. Sempre executar a partir da raiz com `--env-file .env`, como no Makefile. O probe do DB usa TCP (não o socket do servidor temporário de init da imagem oficial). Banco saudável → job migrate completo → API saudável → Vite. A API usa `flagpilot_app`; o job usa `flagpilot_migrator`. A URL host usa loopback; dentro da rede Compose usa `db:5432`.

`make dev-db` sobe apenas PostgreSQL e executa migrations como job descartável. `make dev-containers` executa toda a stack. `make down` encerra os serviços próprios sem remover volume. Não usar `docker system prune` nem `down -v` para solucionar falha de migration.

O script de init do PostgreSQL cria o usuário runtime **somente na criação do volume**. Alterar senha no `.env` não altera automaticamente papel de banco existente. Se precisar rotacionar, executar procedimento SQL controlado sem colocar senha em histórico/log, atualizar `.env` e reiniciar os serviços. Nunca apagar volume com dados úteis para contornar diferença de senha.

`compose.smoke.yml` é exclusivo dos testes, com project name aleatório e portas efêmeras. A limpeza de volume no smoke é intencional e restrita a dados descartáveis próprios.

Dockerfiles têm stages de build e runtime. API é estática, imagem scratch, UID 65532, CA certificates, filesystem read-only no Compose, cap_drop ALL e healthcheck compilado. Web de runtime usa nginx não-root com CSP e proxy de mesma origem; localhost no proxy significa o próprio container, portanto encaminha para `api`. A imagem de desenvolvimento usa Node e Vite; não deve ser publicada como frontend de produção.

Não há compose de produção executável neste incremento. Antes da VPS: confirmar capacidade, versões/digests, banco e segredos separados, TLS banco (verify-full ou decisão de ameaça registrada para rede privada), DNS, regras de proxy que impeçam acesso público a readiness/métricas, backups/restore e autorização externa. A porta da API não deve ser publicada diretamente no destino. Nenhum serviço de Coolify/Traefik ou monitor externo é alterado por estes arquivos.

Desde 2A, a API também contém `/admin` para bootstrap/recuperação em terminal local (`docker compose ... exec api /admin bootstrap`). A atualização preserva volume e aplica migrations até 000003; a API nova exige schema 3. PUBLIC_BASE_URL/ALLOWED_ORIGINS precisam corresponder à origem real do painel. `scripts/auth-e2e.py` usa profile browser de compose.smoke.yml com projeto, volume, portas e conta próprios; não opera no banco de desenvolvimento.
