# ADR-015 — Evidências de imagens e reversão compatível

Data: 25/10/2026. Estado: implementada localmente em 6E; gate de imagens bloqueado.

Fixar bases Go/Node/NGINX e imagens PostgreSQL/migrate por digest, conservando versões nesta etapa para medir a exposição original. `deploy/images.lock.json` registra referências. Não confundir pinning com correção de vulnerabilidades: o scan atual detectou HIGH/CRITICAL em três runtimes, exigindo remediação/triagem explícita antes de release. Não atualizar major do banco ou substituir base/driver durante o ensaio para obter um resultado verde sem validar a mudança.

Usar Trivy 0.69.3 fixado por digest, após consultar advisory upstream do incidente de 2026. Atualizar DB em container sem acesso ao código/socket/segredos; depois analisar arquivos exportados com rede desligada e mounts limitados. Inventário CycloneDX e relatório JSON por runtime; manifest com hashes/IDs e versão/data da DB. Sem dependência nova no produto, scanner global, push ou publicação. Cache e artefatos gerados em `.tools`; resumo sem dados de aplicação no Git. Cotas 1 CPU/1 GiB e temporário em disco evitam pressionar o notebook com uma stack permanente.

Gate fail-closed se inventário/ferramenta falhar; HIGH/CRITICAL bloqueiam inclusive sem versão corrigida, UNKNOWN permanece em triagem. Sem allowlist implícita, supressão de unfixed ou alegação automática de explorabilidade. Os testes de regressão cobrem inventário ausente e bloqueio de HIGH/CRITICAL sem correção. Workflow manual pode guardar artefatos mesmo com falha; nada foi executado remotamente. Assinatura/provenance e digests de publicação continuam pendentes.

Ensaiar rollback de API entre fontes locais 6B e candidata atual, ambas schema 5, em recursos próprios. Histórico é reconstruído com base Go fixada atual, não uma imagem de release preservada. A reversão troca somente binário; preserva DB e todas as tabelas, sessão, revogação, avaliação percentual e escritas, incluindo reaplicação da candidata. Não executar down migration/restore. Esse ensaio não cobre schema incompatível, frontend/proxy nem rollback em produção. Para schema futuro, exigir compatibilidade expand/contract e plano de recuperação antes de tráfego.

Método, limitações e fontes no [runbook](../runbooks/release.md), evidências no [estado de release](../RELEASE_STATUS.md).
