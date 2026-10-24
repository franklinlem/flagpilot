# ADR-002 — Identidade administrativa

Data: 24/10/2026. Estado: Aceita como direção; implementação no incremento 2.

## Contexto

Não há fluxo de primeiro administrador ou recuperação na tabela HTTP original.

## Decisão

Sessão opaca aleatória, hash no banco e revogação; Argon2id para senhas. Bootstrap/recuperação via CLI segura, sem signup público/senha padrão. Cookie HttpOnly/SameSite/Secure em HTTPS, CSRF em mutações, origem validada inclusive login. Recuperação/desativação revoga sessões.

## Alternativas

JWT não foi escolhido por exigir revogação adicional; signup e email transacional não entram no MVP.

## Consequências

Apenas schema de identidade existe agora. CLI bootstrap terá auditoria desde sua primeira entrega; duração absoluta inicial 12 h, a confirmar nos testes/UX de identidade.
