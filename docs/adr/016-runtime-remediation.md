# ADR-016 — Correção dos runtimes sem supressões

Data: 25/10/2026. Estado: implementada localmente em 6F.

Atualizar NGINX para 1.30.6 e PostgreSQL dentro da major 16 para 16.15, com bases fixadas por digest. A atualização isolada de tags ainda deixava gosu antigo e pacotes Alpine afetados: corrigir esses componentes efetivamente. Web usa versões APK explícitas; DB recompila a mesma fonte gosu 1.19 com Go fixado/depêndencias corrigidas em módulo de build separado, mantendo entrypoint/identidade upstream. Regenerar SBOM e verificar conteúdo completo da imagem resultante.

Eliminar a imagem genérica do migrator com múltiplos drivers: wrapper local sobre golang-migrate 4.20.1, somente postgres/file, scratch/nonroot, up-only e configuração por ambiente sem DSN em argv/logs. A biblioteca e modelo de migrations permanecem iguais. CLI host de manutenção é recompilado com Go fixado, interface tradicional e apenas drivers usados. Sem mudanças nas dependências do produto ou na criptografia de senhas.

Manter política do gate e evidência bruta. GO-2026-5932 do módulo x/crypto não tem correção de módulo e se refere a OpenPGP; comprovar ausência desses pacotes/imports/símbolos nos binários da aplicação via go list/govulncheck, sem ocultar UNKNOWN no scanner. Nenhum HIGH/CRITICAL/MEDIUM foi dispensado por suposição de alcance.

Testar upgrade do mesmo volume sintético 16.13→16.15, dump/restore, rollback de API com schema 5, integração e browser com novos runtimes. Atualização local de desenvolvimento mantém volume e conteúdo; cópia prévia privada não é backup recorrente, offsite ou recuperação produtiva. Nenhuma autorização externa é inferida. Condições, fontes e resultados em [SECURITY_REMEDIATION_STATUS.md](../SECURITY_REMEDIATION_STATUS.md).
