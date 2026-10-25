# ADR-003 — Bucketing determinístico v1

Data: 24/10/2026. Estado: Aceita e implementada localmente no incremento 4B.

## Contexto

A especificação define o hash, mas não explicita a redução para 10.000 faixas.

## Decisão

SHA-256 dos componentes separados por dois-pontos na ordem da fonte, UTF-8 e UUIDs canônicos; primeiros 8 bytes uint64 big-endian; bucket=n%10000; enabled e bucket<rollout_bps. Salt hex fixo; JS BigInt; sem normalização Unicode. Somente bucketingKey pode conter delimitador.

## Alternativas

Comparar floats ou converter uint64 para Number perderia precisão; trocar fórmula depois redistribuiria usuários.

## Consequências

Vetores versionados Go/TypeScript/JS em `testdata/bucketing-v1.json`, incluindo buckets 0/9999, Unicode composto/decomposto e mudança de escopo/salt, fixam o contrato. Desativada sempre false, 0% false, 100% true; contexto ausente no parcial retorna false/MISSING_CONTEXT. Booleano vem primeiro.

## Implementação 4B

O avaliador Go lê salt e estado no snapshot protegido pelo lock SHARE do ambiente, sem transportar contexto para SQL/logs/auditoria. UUIDs vêm do banco e são formatados canonicamente. O resultado parcial usa PERCENTAGE_ROLLOUT tanto dentro quanto fora do grupo; MISSING_CONTEXT é reservado à ausência da chave em parcial habilitado. Chave vazia ou acima de 256 bytes UTF-8 é inválida (422). Contexto não é exigido em 0/100% ou flag desativada.

O contrato administrativo aceita qualquer inteiro rolloutBps de 0 a 10000. A UI converte porcentagens com até duas casas decimais exatamente (vírgula ou ponto), sem arredondar silenciosamente. Salvar percentual preserva enabled; ativar/desativar preserva percentual, inclusive 0%. A UI mostra estado e porcentagem separadamente e explica quando nenhum contexto recebe a flag. Produção exige confirmação com valores anteriores/novos. Salt nunca muda nessas operações e não aparece no contrato público ou na auditoria.

Não há migration: schema 5 já suporta pontos-base e salt estável. Revisão/versão/auditoria/If-Match continuam na mesma transação. Alterações de descrição, token ou nome de projeto não participam do hash. Alterar a fórmula exige nova versão explícita e decisão sobre redistribuição, nunca uma troca silenciosa.

O módulo TypeScript é uma referência interna testada nos builds ESM/CJS, fora dos exports públicos; não fornece cliente, snapshot ou avaliação local para consumidores. O SDK HTTP segue para a etapa 5. O percentual representa distribuição aproximada de contextos, sem quota exata; aumentar somente a porcentagem mantém os contextos já incluídos, enquanto diminuir remove apenas os buckets acima do novo limiar.
