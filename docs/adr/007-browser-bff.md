# ADR-007 — Browser via BFF

Data: 24/10/2026. Estado: Aceita como direção; implementação junto ao SDK.

## Contexto

Token de avaliação secreto não pode estar em bundle público ou localStorage.

## Decisão

Node usa Bearer no servidor. Browser usa BFF da aplicação consumidora, que mantém segredo e restringe flags/usuários e contexto. Adapter browser separado da opção token; endpoint publicável/chave pública adiado.

## Alternativas

Chave publicável exigiria classificação de flags, endpoint público e política de abuso próprios.

## Consequências

Nenhum token/browser client nesta fundação. Antes do SDK documentar contrato do adapter e exemplo seguro, sem construir proxy aberto.
