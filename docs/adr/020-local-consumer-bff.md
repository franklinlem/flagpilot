# ADR-020 — Exemplo consumidor local com BFF

Data: 27/10/2026. Estado: implementada (8C), complementa ADR-007.

Exemplo Node com HTTP nativo e página HTML/JS sem build adicional. O browser solicita uma cotação; o backend valida quantidade, consulta uma única flag configurada no servidor e calcula preços/frete em centavos. Token e contexto permanecem no processo Node. Nenhum endpoint de avaliação genérico é disponibilizado.

Flag `shipping_v2`: padrão cobra R$ 12; nova experiência cobra R$ 8 ou frete grátis com 3+ itens. Fallback false conserva regra padrão. Cache desativado no exemplo para visualizar mudanças/revogação imediatamente, timeout 1 s e sem retry. Em integrações reais, escolher TTL conscientemente.

Bind fixo em 127.0.0.1, Host exato, Origin obrigatório no POST, JSON limitado a 1 KiB, quantidade inteira 1..10, CSP e assets allowlisted. Contexto sintético fixo é explicitamente demonstração, sem contas/pedidos/pagamentos. Não é autenticação de aplicação real. Um BFF implantado deve derivar contexto da sessão validada, aplicar autorização independente da flag e limites adequados; não publicar este servidor demonstrativo.

Sem Express, React/Vite adicional, dependências de runtime, migration ou alteração na API central. Teste com API/DB descartáveis, browser Chromium e parada real da API cobre alternância, rollout e fallback, além das validações locais.
