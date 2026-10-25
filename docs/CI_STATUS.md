# Integração com GitHub

25/10/2026: repositório privado franklinlem/flagpilot criado e histórico local enviado, com scan Gitleaks completo aprovado. Nenhuma licença pública foi escolhida automaticamente.

Primeira execução real de CI passou por geração/lint/tipos, Go/race, integração PostgreSQL, builds, SCA e recuperação, mas o preparo de browser tentou usar uma porta efêmera antiga depois de recriar web. Corrigido o harness para preservar a porta atribuída. No Firefox local apareceu também uma corrida de clique enquanto a lista inicial de projetos mudava o layout; o teste agora aguarda o estado de carregamento terminar, sem retries ou aumento de timeout. Seis fluxos Firefox passaram após as correções. Nova execução remota será acompanhada; não marcar a CI aprovada antes do resultado.

Publicação de imagens/npm, releases públicas e deploy automático continuam desabilitados. O envio ao GitHub foi autorizado explicitamente por Franklin; a implantação externa é uma homologação isolada autorizada no domínio flagpilot.franklem.uk.
