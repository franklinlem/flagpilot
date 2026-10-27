# Integração com GitHub

25/10/2026: repositório privado franklinlem/flagpilot criado e histórico local enviado, com scan Gitleaks completo aprovado. Nenhuma licença pública foi escolhida automaticamente.

Primeira execução real de CI passou por geração/lint/tipos, Go/race, integração PostgreSQL, builds, SCA e recuperação, mas o preparo de browser tentou usar uma porta efêmera antiga depois de recriar web. Corrigido o harness para preservar a porta atribuída. No Firefox local apareceu também uma corrida de clique enquanto a lista inicial de projetos mudava o layout; o teste agora aguarda o estado de carregamento terminar, sem retries ou aumento de timeout. Seis fluxos Firefox passaram após as correções. A execução seguinte revelou outra corrida no teste ao limpar filtros e editar o formulário antes de a navegação terminar; o teste agora aguarda o catálogo sem filtros aparecer. A execução remota [40214673901](https://github.com/franklinlem/flagpilot/actions/runs/40214673901), revisão `481f112`, concluiu com sucesso, incluindo Chromium e Firefox. A configuração operacional 7A (`65db966`) também passou: [execução 40215731673](https://github.com/franklinlem/flagpilot/actions/runs/40215731673).

Publicação de imagens/npm, releases públicas e deploy automático continuam desabilitados. O envio ao GitHub foi autorizado explicitamente por Franklin; a implantação externa é uma homologação isolada autorizada no domínio flagpilot.franklem.uk.

## Continuidade 8A

A execução 40531365526 (7B) foi cancelada pela concorrência do workflow quando a revisão funcional seguinte foi enviada, sem conclusão dos browsers. Seus logs mostram o download APT de 53,7 MB pelo mirror azure.archive.ubuntu.com em 20min32s, antes de iniciar o primeiro fluxo. Não foi falha de asserção da aplicação. O workflow agora troca somente esse endereço pelo arquivo oficial HTTPS archive.ubuntu.com no runner efêmero, preservando suites, Signed-By e validação de pacotes. Formatos de configuração conforme [documentação Ubuntu](https://ubuntu.com/server/docs/how-to/software/package-management/). Nenhuma alteração de APT no notebook/VPS/host físico.

A suíte de segurança da conta tem fixture própria para não consumir o orçamento de login da suíte administrativa; CI roda ambas em Chromium/Firefox, sem retries ou relaxamento de limites. Resultado remoto da revisão final será acompanhado; make ci e verificações locais não são apresentados como CI remota concluída.

Fechamento local 8A: make ci aprovado; 24 testes frontend, 16 SDK, Go/race/vet, integração PostgreSQL com concorrência/rollback, tipos/contrato/builds. Fluxo de segurança em Chromium/Firefox aprovado, seis fluxos administrativos Chromium aprovados; Chromium final também cobre o novo filtro de auditoria e runtime corrigido. Scans finais aprovados. A CI remota final é uma verificação adicional em andamento, não declarada aprovada.

## Fechamento 8A e continuidade 8B–8D

A execução final [40538304391](https://github.com/franklinlem/flagpilot/actions/runs/40538304391), ae94310, concluiu com sucesso; o acompanhamento acima era snapshot anterior. As quatro suítes de browser e demais etapas passaram no GitHub.

8B–8D acrescentam sete testes Python de supervisão de backup, dois testes Node do consumidor e ensaio com BFF/API/DB/Chromium reais, sem dependências novas. CI incorpora os novos testes e o exemplo ao ESLint. O ensaio consumidor tem stack própria, não usa credenciais do servidor e não interrompe homologação. Resultado remoto da revisão nova será registrado após execução; não presumir verde da base.
