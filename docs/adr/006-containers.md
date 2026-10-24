# ADR-006 — Containers e operação

Data: 24/10/2026. Estado: Aceita; desenvolvimento iniciado, destino não alterado.

## Contexto

Debian local e VPS Coolify precisam de artefatos reproduzíveis sem alterar serviços existentes.

## Decisão

Modo host leve e Compose completo alternativo. PostgreSQL 16.13 em rede própria; migrate como job separado e usuário runtime sem DDL. API multi-stage/scratch não-root com CA e healthcheck compilado; frontend estático nginx não-root/proxy mesma origem. Sem CD nesta etapa.

## Alternativas

Traefik local e stack obrigatória de observabilidade não são necessários. Auto-migrate em cada réplica traz concorrência/DDL ao runtime.

## Consequências

Init PostgreSQL local usa papel superuser da imagem oficial, nunca compartilhado com API. No destino provisionar migrador sem superuser e runtime separados. TLS PostgreSQL verify-full requerido em modo production; se a topologia usar rede privada sem TLS, revisar explicitamente esta política antes de implantar. Imagens de release por digest; aprovação externa futura obrigatória.
