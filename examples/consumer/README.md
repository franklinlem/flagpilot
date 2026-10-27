# Cotação com FlagPilot — consumidor local

Exemplo de BFF Node 24 + SDK privado. Uma flag altera uma função real de cotação, mantendo cálculo e token no backend. Sem pedidos, pagamento, dados pessoais ou autenticação de clientes reais. Bind somente em 127.0.0.1; não publicar este servidor demonstrativo.

Na raiz do monorepo, com `npm ci` já executado:

```sh
cp examples/consumer/.env.example examples/consumer/.env
chmod 600 examples/consumer/.env
# Editar .env no VS Code: URL, token de avaliação e chave/contexto sintético.
npm run demo:consumer
```

Abra **http://127.0.0.1:3100** (usar esse endereço exato). Para a API local seguir README principal; para homologação usar `https://flagpilot.franklem.uk/api/v1`. Emita token de avaliação no ambiente de teste, com expiração curta, e guarde somente no `.env` ignorado; revogue após o uso. Nunca usar token administrativo nem VITE_*.

1. Crie a flag **shipping_v2** no projeto. Ela pode existir desativada.
2. Com quantidade 3, regra padrão resulta em produtos R$ 60 + frete R$ 12 = R$ 72.
3. Ative no ambiente do token com rollout 100%: mesmos itens resultam em R$ 60, frete grátis. Com 1 item, nova regra resulta em R$ 28.
4. Rollout 0% volta ao padrão. Percentual parcial mantém resultado estável para o mesmo contexto; ele não alterna a cada clique. Para outra identidade sintética, edite FLAGPILOT_BUCKETING_KEY no servidor e reinicie.
5. Revogue o token ou pare a API **local descartável**, sem interromper homologação: experiência padrão permanece e a página informa fallback. TTL do exemplo é zero, timeout 1 s e sem retry.
6. Ctrl+C encerra o consumidor e fecha o SDK. `.env` nunca é servido; browser chama apenas `/api/quote`, sem flags/contextos arbitrários ou Bearer.

O browser só envia quantidade inteira de 1 a 10. Não aceitar preço/flag/contexto fornecido pelo browser. Em sistema real, derive identidade da sessão autenticada, mantenha autorização independente das flags e limite abuso. Esta cotação local não implementa autenticação, cobrança, persistência ou contabilidade.

```sh
npm run test:consumer
python3 scripts/consumer-e2e.py
```

O segundo comando exige Docker/Compose e Chromium instalado (`PLAYWRIGHT_BROWSERS_PATH="$PWD/.tools/playwright" npx playwright install chromium`). Cria API/DB/conta/token descartáveis, testa UI com 0%, 100%, 37,25% e flag desligada, para a API própria para verificar fallback e remove seus recursos. Não usa banco de desenvolvimento ou credenciais reais. Evidência em `docs/evidence/consumer-2026-10-27.json` é um snapshot do ensaio. Para registrar outro, forneça `--report CAMINHO_NOVO.json`; relatórios existentes são recusados antes de criar recursos. Sem essa opção, não altera evidências versionadas. O harness não publica nada.
