# ADR-002 — Identidade administrativa

Data: 24/10/2026. Estado: Aceita; identidade entregue no incremento 2A.

## Contexto

Não há fluxo de primeiro administrador ou recuperação na tabela HTTP original.

## Decisão

Sessão opaca aleatória, hash no banco e revogação; Argon2id para senhas. Bootstrap/recuperação via CLI segura, sem signup público/senha padrão. Cookie HttpOnly/SameSite/Secure em HTTPS, CSRF em mutações, origem validada inclusive login. Recuperação/desativação revoga sessões.

## Alternativas

JWT não foi escolhido por exigir revogação adicional; signup e email transacional não entram no MVP.

## Consequências

Bootstrap/recuperação CLI e login/logout/me estão implementados com auditoria desde a primeira entrega. Duração absoluta inicial 12 h, configurável entre 1h e 168h. CSRF via HMAC da sessão, sem SESSION_SECRET/CSRF_SECRET redundantes; email/senha nunca entram nos logs HTTP. Argon2id 19 MiB/2 iterações/p=1; no máximo duas verificações simultâneas por processo. Prompt oculto com restauração do terminal em SIGINT no Linux. Ver runbook de identidade para limites e operação.
