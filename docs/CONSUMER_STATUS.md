# Aplicação consumidora — incremento 8C

27/10/2026. Exemplo local em [examples/consumer](../examples/consumer/README.md), sem novas dependências de runtime, sem deploy externo e sem mudança no schema/API central.

BFF Node usa SDK para calcular cotação e frete no servidor. Browser envia apenas quantidade; token, flag allowlisted e contexto sintético permanecem no Node. Validação de entrada, origem/Host, payload limitado, CSP e fallback padrão. Contexto fixo de demonstração; autenticação/autorização de uma aplicação real não estão implementadas neste exemplo e não podem ser substituídas pela flag. [ADR-020](adr/020-local-consumer-bff.md).

[Evidência](evidence/consumer-2026-10-27.json): API/PostgreSQL reais em stack descartável, Chromium headless, flag desligada, rollout 100%, 0%, 37,25% e fallback após parar a API própria. A cotação mudou entre R$ 72 e R$ 60 para três itens; resultado parcial comparado à avaliação direta da API. Browser chamou somente BFF/assets da mesma origem; token ausente nos headers/payloads/respostas observados e storage vazio. Recursos próprios removidos.

Dois testes Node adicionais cobrem cálculo, contexto fixo, entradas extras/inválidas, origem/content-type/payload, assets e SDK real com transporte simulando erro de rede, 401 e timeout. São ensaios automatizados; aceitação visual humana do exemplo ainda não ocorreu. Nenhum ensaio usa conta/token de Franklin ou interrompe a homologação.

Executar: `npm run demo:consumer`, após preparar `.env` privado conforme README do exemplo. Verificar: `npm run test:consumer` e `python3 scripts/consumer-e2e.py`. CI inclui ambos; [execução remota 40576982060](https://github.com/franklinlem/flagpilot/actions/runs/40576982060), 88dbb04, aprovada com o novo consumidor e suítes centrais Chromium/Firefox. O servidor só escuta em 127.0.0.1 e não deve ser publicado como loja/BFF produtivo.
