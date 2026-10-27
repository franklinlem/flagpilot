# Segurança da conta — incremento 8A

27/10/2026. Próximo incremento funcional autorizado após 7B. Troca de senha do próprio usuário implementada em backend, OpenAPI e painel; recuperação administrativa continua CLI. Sem migration, dependência nova ou alteração de senha real durante os testes.

Conta → Segurança da conta permite informar senha atual, nova e confirmação. Ao concluir, todas as sessões são revogadas e o painel volta ao login com mensagem de sucesso. Política existente: mínimo de 12 caracteres e máximo de 256 bytes UTF-8, espaços preservados, nova diferente da atual. Papéis Viewer/Admin/Owner não limitam o cuidado com a própria credencial; não há parâmetro para trocar senha de terceiros.

POST /api/v1/auth/password exige origem, cookie e CSRF, corpo estrito de até 4 KiB, limitador próprio e vagas Argon2 compartilhadas com login. Senha/hash/sessão nunca entram no JSON de resposta, logs ou auditoria. Atualização/revogação/auditoria organizacional atômicas; lock do usuário compatível com login/recuperação e revalidação da sessão dentro da transação. Decisões/limites em [ADR-018](adr/018-self-service-password.md).

## Verificação

`make ci` completo aprovado: Go/race/vet, integração PostgreSQL, 24 testes frontend, 16 SDK, tipos/lint/OpenAPI e builds. A integração cobre origem/CSRF/sessão, payload/limites, política Unicode/bytes, senha incorreta, rollback quando auditoria falha, revogação de duas sessões, senha antiga rejeitada, auditoria sem segredos, concorrência entre trocas e logout durante a troca. O lock de usuário usa FOR NO KEY UPDATE para não bloquear a FK de auditoria de logout enquanto aguarda a sessão.

Formulário validado para confirmação, limpeza após falhas, sessão expirada e sucesso. Fluxo real novo passou em Chromium e Firefox, em stacks descartáveis separadas da suíte administrativa, mantendo os limites ativos. Seis fluxos administrativos existentes também passaram no Chromium. O fluxo de senha verifica largura de 390 px, foco/título, duas sessões encerradas, senha antiga recusada, novo login, ausência de storage persistente e auditoria sem senhas. Suíte administrativa Firefox da revisão 8A não foi repetida localmente; CI configurada para ambas as suítes em ambos os browsers.

Nenhuma senha de Franklin foi usada ou alterada nesses testes. Aceitação visual humana do novo formulário ainda pendente. A homologação foi atualizada depois do gate e da verificação final do browser, conforme fechamento abaixo.

## Uso

Depois de disponibilizar a revisão desejada, entrar no painel, abrir Segurança da conta, preencher os três campos e escolher Alterar senha e sair. Entrar novamente com a nova senha. As outras sessões deixam de funcionar na próxima consulta; não há logout por push. Tokens de avaliação compartilhados permanecem válidos até sua própria revogação/expiração.

Se perder a senha, usar [recuperação via CLI](runbooks/identity.md); não há recuperação por email nesta etapa. Senha incorreta preserva a sessão, erros limpam os campos. Uma resposta perdida pode ocorrer depois de a senha já ter sido alterada; verificar login com a nova senha antes de repetir. O arquivo privado de senha inicial não é atualizado pelo formulário; após escolher senha própria, guardar no gerenciador de senhas e tratar a cópia inicial como obsoleta.

Próximas frentes, ainda planejadas: [ROADMAP.md](ROADMAP.md).

## Gate de runtime

O primeiro scan de 27/10 bloqueou a candidata: CVE-2026-92392 apareceu como HIGH em curl/libcurl 8.22.0-r0 (duas ocorrências), e CVE-2026-92388 como MEDIUM nesses pacotes; [registro inicial](evidence/images-account-2026-10-27.json). A classificação registrada é a do scanner, sem alterar severidades por inferência de alcance. O fornecedor confirma correção em 8.23.0: [advisory HTTP/2](https://curl.se/docs/CVE-2026-92392.html), [advisory FTP](https://curl.se/docs/CVE-2026-92388.html). O repositório Alpine 3.24 consultado oferece 8.23.0-r0; Dockerfile agora fixa curl/libcurl nessa versão, preservando usuário não root e demais versões. Repetir SBOM/scan e verificar runtime antes de aplicar. Nenhum gate foi relaxado ou aviso suprimido.

Scan corrigido aprovado: [inventário final](evidence/images-account-hardened-2026-10-27.json), zero HIGH/CRITICAL nos quatro runtimes candidatos, web/DB/migrator sem achados e API com os dois UNKNOWN OpenPGP já triados em 6F. Não é garantia de ausência absoluta de vulnerabilidades; nenhuma supressão criada.

## Homologação atualizada

Disponível em https://flagpilot.franklem.uk/account/security, após login. API/web da revisão funcional `c4ca9a2` transportados por SSH e image IDs comparados com o [scan final](evidence/images-account-final-2026-10-27.json), novamente aprovado após incluir o filtro auth.password_changed no painel de auditoria. Fluxo Chromium final repetido com pacotes corrigidos e filtro de auditoria aprovado; lint/tipos finais passaram. Os resultados anteriores de Firefox cobrem o formulário/alteração; o novo item estático de filtro não foi repetido localmente no Firefox. CI remota contempla o fluxo completo nos dois browsers, mas seu resultado não é presumido.

Backup criptografado adicional obtido no frank-note2 antes da atualização (`flagpilot-homolog-20261027T155717Z.dump.gpg`). Somente API/web recriados, sem build/pull remoto; DB/migrator/proxy e demais containers preservados. Schema continua 5, sem migration. Configuração persistida no Coolify mantém rede de ingresso coolify, sem depender da conexão transitória do proxy com a rede exclusiva.

[Evidência externa](evidence/account-homolog-2026-10-27.json): ambos os runtimes saudáveis e com os image IDs esperados, HTML público igual ao arquivo da imagem implantada, HTTPS/API 200 inclusive a partir do frank-note2, endpoint de senha sem sessão 401 e endpoints internos 404. curl/libcurl 8.23.0-r0 confirmados no container web. Monitores 14/15 UP, canal autorizado preservado. Nenhuma senha pessoal de Franklin foi usada ou alterada automaticamente; a validação de troca foi feita com contas sintéticas descartáveis. Credencial temporária de administração revogada após conferência.

A validação manual do novo formulário continua pendente. A senha inicial privada é obsoleta depois de o próprio usuário escolher outra; não sincronizamos uma senha pessoal com o arquivo de bootstrap. Produção, licença pública e publicação de pacotes/imagens não foram alteradas.
