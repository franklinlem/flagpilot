# Ensaio local de backup e restauração

Este procedimento usa SOMENTE dados sintéticos e recursos descartáveis. Não aceita URL de banco, projeto Compose existente ou dump externo. Não configura backup recorrente nem toca o volume flagpilot-dev.

```bash
make recovery-rehearsal
# Para guardar somente evidências sem dados/credenciais, escolha um arquivo novo:
python3 scripts/recovery.py --report /tmp/flagpilot-recovery-report.json
```

Requer Docker/Compose local por socket Unix; recusa host/contexto Docker remoto e ignora overrides de credenciais/projeto do shell para usar somente o .env sintético. Usa a imagem local corrigida baseada em PostgreSQL 16.15, com gosu recompilado, e a API construída do checkout. O projeto aleatório flagpilot-recovery-* é impresso ao iniciar. Cada execução cria suas próprias senhas em .env temporário 0600 e diretório 0700. O arquivo de dump é custom (-Fc), comprimido, sem criptografia, de fixtures sintéticas, 0600; é eliminado no fim. Não usar esse tratamento temporário como política de proteção de backups reais.

## Upgrade sintético opcional (6F)

```bash
python3 scripts/recovery.py --upgrade-from-16-13 --report docs/evidence/recuperacao-upgrade-novo.json
```

A origem usa a imagem histórica 16.13 fixada apenas em recursos descartáveis. Após criar fixture, para API, troca DB para o build local 16.15 no mesmo volume e compara todas as tabelas/avaliação antes de seguir com dump/restore. Não usa dados reais. Ver recovery-hardened-2026-10-25.json; as referências 6C abaixo são evidência histórica anterior. Extensões especiais/replicação não estão na fixture; revisar notas de release antes de bancos reais.

## O que o ensaio comprova

1. Inicializa schema pelas migrations e cria usuário, organização, projeto, ambientes, flags, configuração parcial, token ativo/revogado e sessão pelo fluxo da aplicação.
2. Calcula contagem/hash canônico de cada uma das 11 tabelas em memória, sem imprimir linhas ou hashes de credenciais.
3. Executa pg_dump --format=custom; depois revoga o token ativo para simular uma mudança posterior ao snapshot.
4. Remove a origem e seus volumes; verifica que não restou volume com o label do projeto antes de criar o destino.
5. Restaura em banco novo com pg_restore --no-owner --exit-on-error --single-transaction. Roles migrator/runtime são provisionadas no destino com novas senhas, pois pg_dump não inclui roles globais. Grants/ACLs do dump são mantidos.
6. Confere todas as tabelas antes de qualquer ajuste de segurança. Revoga todas as sessões e tokens restaurados e registra recovery.credentials_invalidated por operador local, em transação, antes de iniciar a API restaurada.
7. Confirma readiness, rejeição da sessão/tokens antigos, login com a senha da conta de teste, emissão de credencial nova, mesma avaliação/revisão e runtime sem DDL.
8. Remove somente os recursos próprios do projeto e apaga o dump/.env temporários. O relatório opcional contém apenas contagens, tempos, tamanho e resultados booleanos, sem dados do banco.

O ensaio usa banco pequeno. Tempos registrados são evidência daquela fixture/notebook, não RPO/RTO de produção nem capacidade do backup. Uma execução pontual não cria agendamento, retenção ou cópia externa. Ver [evidência local](../evidence/recovery-2026-10-25.json).

## Política para planejar recuperação real

O dump representa o estado do snapshot: restaurá-lo pode reativar sessões/tokens revogados depois da coleta. Antes de liberar tráfego real, a recuperação deve invalidar credenciais restauradas em ambiente isolado, auditar a operação e reemitir tokens de consumidores. Alterações posteriores ao snapshot podem ser perdidas; revisar também senhas/contas conforme o incidente. O teste invalida sessões/tokens, sem alegar que troca automaticamente senhas humanas.

Uma rotina real ainda precisa de artefatos criptografados, proteção/recuperação das chaves, retenção, cópia externa, verificação de integridade, alertas de falha e ensaios periódicos. Uma restauração real precisa de inventário, revisão do dump confiável, espaço/capacidade, versão compatível, roles/privilégios, isolamento de tráfego e autorização específica. O script sintético não aceita esses destinos justamente para evitar confundir ensaio com operação real.

Para interromper, use Ctrl+C; o finally tenta limpar os recursos próprios. Se a limpeza falhar, identifique o projeto aleatório impresso e inspecione somente seus labels com Docker. Não use down --volumes no projeto de desenvolvimento para reproduzir esse teste. Relatórios não sobrescrevem arquivos existentes.

Base técnica: [pg_dump PostgreSQL 16](https://www.postgresql.org/docs/16/app-pgdump.html) e [pg_restore PostgreSQL 16](https://www.postgresql.org/docs/16/app-pgrestore.html).
