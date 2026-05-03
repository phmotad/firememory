# FireMemory + FireQuery â€” Tasks

## Parte 1 â€” FireMemory Core

### Fase 0 â€” Setup

- [x] Criar repositÃ³rio
- [x] Criar `go.mod`
- [x] Criar estrutura de pastas
- [x] Criar `README.md`
- [x] Criar `AGENTS.md`
- [x] Criar `CLAUDE.md`
- [x] Criar `LICENSE`
- [x] Criar pasta `docs`
- [x] Criar pasta `examples`
- [x] Definir `.fbrain` como extensÃ£o oficial

### Fase 1 â€” EspecificaÃ§Ã£o

- [x] Criar `docs/vision.md`
- [x] Criar `docs/architecture.md`
- [x] Criar `docs/domain.md`
- [x] Criar `docs/brainfile-format.md`
- [x] Criar `docs/embedding.md`
- [ ] Criar `docs/dedup-engine.md`
- [ ] Criar `docs/context-engine.md`
- [x] Criar `docs/mcp-integration.md`
- [ ] Criar `docs/adp-protocol.md`
- [x] Criar `docs/roadmap.md`
- [x] Criar `docs/tasks.md`

### Fase 2 â€” DomÃ­nio

- [x] Criar `Memory`
- [x] Criar `Entity`
- [x] Criar `Relation`
- [x] Criar `Fact`
- [x] Criar `Event`
- [x] Criar `Concept`
- [x] Criar `SourceRef`
- [x] Criar `MemoryKind`
- [x] Criar `MemoryStatus`
- [x] Criar `DedupAction`
- [x] Criar `RelationType`
- [x] Criar inputs/outputs da Engine
- [x] Criar testes dos tipos

### Fase 3 â€” Storage Interface

- [x] Criar interface `Store`
- [x] Criar interface `Tx`
- [x] Criar `Put`
- [x] Criar `Get`
- [x] Criar `Delete`
- [x] Criar `List`
- [x] Criar `View`
- [x] Criar `Update`
- [x] Criar `Snapshot`
- [x] Criar `Compact`
- [x] Criar fake store para testes

### Fase 4 â€” BboltStore

- [x] Adicionar `bbolt`
- [x] Implementar `BboltStore`
- [x] Implementar namespaces internos
- [x] Implementar Open
- [x] Implementar Close
- [x] Implementar Put/Get/Delete/List
- [x] Implementar transaÃ§Ãµes
- [x] Implementar Snapshot
- [x] Criar stub de Compact
- [x] Garantir que bbolt nÃ£o aparece na API pÃºblica
- [x] Criar testes de persistÃªncia

### Fase 5 â€” Brainfile `.fbrain`

- [x] Criar `BrainManifest`
- [x] Criar arquivo `.fbrain`
- [x] Validar extensÃ£o `.fbrain`
- [x] Persistir manifest
- [x] Ler manifest
- [x] Validar versÃ£o do formato
- [x] Criar namespaces:
  - manifest
  - memories
  - entities
  - relations
  - facts
  - events
  - concepts
  - sources
  - vectors
  - graph_nodes
  - graph_edges
  - hash_index
  - traces
  - sync_queue
- [x] Criar `inspect`
- [x] Criar testes

### Fase 6 â€” Embedder

- [x] Criar interface `Embedder`
- [x] Criar `DeterministicEmbedder`
- [x] Criar `ExternalEmbedder`
- [x] Criar `E5Embedder`
- [x] Documentar `intfloat/multilingual-e5-small`
- [x] Implementar normalizaÃ§Ã£o L2
- [x] Validar dimensÃ£o
- [x] Criar testes

### Fase 7 â€” Vector Engine

- [x] Criar `VectorIndex`
- [x] Criar `LinearVectorIndex`
- [x] Implementar Add
- [x] Implementar Search
- [x] Implementar Remove
- [x] Implementar cosine similarity
- [x] Implementar filtros por scope
- [x] Implementar filtros por kind
- [x] Implementar topK
- [x] Criar testes

### Fase 8 â€” Graph Engine

- [x] Criar `Node`
- [x] Criar `Edge`
- [x] Criar interface `Graph`
- [x] Implementar AddNode
- [x] Implementar AddEdge
- [x] Implementar Neighbors
- [x] Implementar Related
- [x] Implementar traversal por profundidade
- [x] Persistir nÃ³s
- [x] Persistir arestas
- [x] Criar testes

### Fase 9 â€” Dedup Fast Path

- [x] Criar normalizador textual
- [x] Criar hash normalizado
- [x] Criar `DedupResult`
- [x] Criar dedup exato
- [x] Criar dedup vetorial
- [x] Criar decisÃ£o `create_new`
- [x] Criar decisÃ£o `reinforce`
- [x] Criar testes

### Fase 10 â€” Engine Base

- [x] Criar interface `Engine`
- [x] Criar `Options`
- [x] Implementar Open
- [x] Implementar Close
- [x] Carregar manifest
- [x] Reconstruir Ã­ndices
- [x] Injetar Store
- [x] Injetar Embedder
- [x] Injetar VectorIndex
- [x] Injetar Graph
- [x] Criar testes

### Fase 11 â€” Remember

- [x] Criar `RememberInput`
- [x] Criar `RememberResult`
- [x] Validar input
- [x] Normalizar conteÃºdo
- [x] Gerar hash
- [x] Gerar embedding
- [x] Executar dedup fast path
- [x] Criar memÃ³ria
- [x] Persistir memÃ³ria
- [x] Atualizar vetor
- [x] Marcar `pending_sync`
- [x] Retornar trace
- [x] Criar testes

### Fase 12 â€” Recall

- [x] Criar `RecallInput`
- [x] Criar `RecallResult`
- [x] Gerar embedding da query
- [x] Buscar vetorialmente
- [x] Buscar lexicalmente
- [x] Combinar resultados
- [x] Aplicar ranking
- [x] Retornar trace
- [x] Criar testes

### Fase 13 â€” Extractor

- [x] Criar interface `Extractor`
- [x] Criar `ExtractionResult`
- [x] Criar `ExtractedEntity`
- [x] Criar `ExtractedFact`
- [x] Criar `HeuristicExtractor`
- [x] Extrair versÃµes
- [x] Extrair possÃ­veis nomes prÃ³prios
- [x] Extrair termos tÃ©cnicos
- [x] Extrair palavras-chave
- [x] Criar `GLiNERExtractor` opcional
- [x] Criar testes

### Fase 14 â€” Relation Classifier

- [x] Criar `MemoryRelationClassifier`
- [x] Classificar duplicate
- [x] Classificar reinforce
- [x] Classificar complement
- [x] Classificar update
- [x] Classificar conflict
- [x] Implementar versÃ£o heurÃ­stica
- [x] Criar testes

### Fase 15 â€” Sync Slow Path

- [x] Criar `SyncInput`
- [x] Criar `SyncResult`
- [x] Buscar memÃ³rias `pending_sync`
- [x] Executar extractor
- [x] Criar entidades
- [x] Criar fatos
- [x] Criar relaÃ§Ãµes
- [x] Classificar relaÃ§Ãµes entre memÃ³rias prÃ³ximas
- [x] Atualizar grafo
- [x] Atualizar importÃ¢ncia/confianÃ§a
- [x] Marcar `synced`
- [x] Retornar relatÃ³rio
- [x] Criar testes

### Fase 16 â€” Context Engine

- [x] Criar `ContextInput`
- [x] Criar `ContextResult`
- [x] Buscar memÃ³rias relevantes
- [x] Expandir grafo
- [x] Incluir entidades
- [x] Incluir relaÃ§Ãµes
- [x] Incluir fatos
- [x] Aplicar ranking
- [x] Estimar tokens
- [x] Respeitar budget
- [x] Montar contexto final
- [x] Retornar trace
- [x] Criar testes

### Fase 17 â€” Explain

- [x] Criar `ExplainInput`
- [x] Criar `ExplainResult`
- [x] Explicar recall
- [x] Explicar dedup
- [x] Explicar contexto
- [x] Criar testes

### Fase 18 â€” CLI `fmem`

- [x] Implementar `fmem init`
- [x] Implementar `fmem remember`
- [x] Implementar `fmem recall`
- [x] Implementar `fmem sync`
- [x] Implementar `fmem context`
- [x] Implementar `fmem inspect`
- [x] Implementar `fmem snapshot`
- [x] Implementar `fmem compact`
- [x] Validar extensÃ£o `.fbrain`
- [x] Criar saÃ­da legÃ­vel
- [x] Testar comandos

### Fase 19 â€” MCP FireMemory

- [x] Criar docs MCP
- [x] Criar stub MCP
- [x] Definir tools:
  - firememory.remember
  - firememory.recall
  - firememory.get_context
  - firememory.sync
  - firememory.explain
- [x] Criar schemas
- [x] Criar testes bÃ¡sicos

### Fase 20 â€” Exemplos

- [x] Criar `examples/basic`
- [x] Criar `examples/support-agent`
- [x] Criar `examples/fireagent-integration`
- [x] Criar demo end-to-end
- [x] Atualizar README

### Fase 21 â€” Aceite FireMemory

- [x] `go test ./...` passa
- [x] `fmem init ./agent.fbrain` funciona
- [x] `fmem remember` funciona
- [x] `fmem recall` funciona
- [x] `fmem sync` funciona
- [x] `fmem context` funciona
- [x] Dados persistem apÃ³s reabrir
- [x] `.fbrain` Ã© arquivo Ãºnico
- [x] bbolt nÃ£o aparece na API pÃºblica

---

## Parte 2 â€” FireQuery Complete

SÃ³ iniciar depois da Parte 1 concluÃ­da.

### Fase 22 â€” FireQuery Spec

- [x] Criar `docs/firequery-architecture.md`
- [x] Criar `docs/firequery-contract.md`
- [x] Criar `docs/firequery-models.md`
- [x] Criar `docs/firequery-mcp.md`
- [x] Criar `docs/firequery-runtime.md`
- [x] Definir contrato externo Agente â†’ FireQuery
- [x] Definir contrato interno FireQuery â†’ FireMemory

### Fase 23 â€” Estrutura FireQuery

- [x] Criar `internal/firequery`
- [x] Criar `contract`
- [x] Criar `validator`
- [x] Criar `pipeline`
- [x] Criar `models`
- [x] Criar `adapters`
- [x] Criar `runtime`
- [x] Criar `mcp`
- [x] Criar `doctor`

### Fase 24 â€” Contract Validator

- [x] Criar `OperationRequest`
- [x] Criar `OperationResponse`
- [x] Exigir `language = "en"`
- [x] Exigir `version`
- [x] Exigir `request_id`
- [x] Exigir `actor`
- [x] Exigir `operation`
- [x] Exigir `intent`
- [x] Exigir `brain`
- [x] Exigir `scope`
- [x] Validar thresholds
- [x] Validar permissÃµes
- [x] Rejeitar writes invÃ¡lidos
- [x] Rejeitar forget sem confirmaÃ§Ã£o
- [x] Garantir que request invÃ¡lida nÃ£o chama FireMemory
- [x] Criar testes

### Fase 25 â€” Especialistas

- [x] Criar `IntentClassifier`
- [x] Criar `TriggerClassifier`
- [x] Criar `EntityExtractor`
- [x] Criar `FactExtractor`
- [x] Criar `RelationClassifier`
- [x] Criar `SimilarityEngine`
- [x] Criar `Reranker`
- [x] Criar interfaces
- [x] Criar adapters
- [x] Criar versÃµes mock para testes

### Fase 26 â€” Model Runtime

- [x] Criar `DeviceDetector`
- [x] Criar `BackendSelector`
- [x] Suportar CUDA
- [x] Suportar DirectML
- [x] Suportar CoreML
- [x] Suportar OpenVINO
- [x] Suportar CPU fallback
- [x] Criar lazy loading
- [x] Criar memory budget
- [x] Criar model health check
- [x] Criar `fquery devices`
- [x] Criar `fquery doctor`

### Fase 27 â€” FireQuery Pipeline

- [x] Receber request MCP
- [x] Validar contrato externo
- [x] Classificar intenÃ§Ã£o
- [x] Detectar trigger
- [x] Extrair entidades
- [x] Extrair fatos
- [x] Buscar similaridade
- [x] Classificar relaÃ§Ã£o
- [x] Reranqueamento
- [x] Montar contrato interno
- [x] Validar contrato interno
- [x] Chamar FireMemory
- [x] Retornar resposta estruturada
- [x] Criar testes end-to-end

### Fase 28 â€” MCP FireQuery

- [x] Criar tools:
  - firequery.ask
  - firequery.plan
  - firequery.remember
  - firequery.recall
  - firequery.get_context
  - firequery.explain
- [x] Criar schemas
- [x] Criar exemplos
- [x] Criar documentaÃ§Ã£o

### Fase 29 â€” Aceite FireQuery

- [x] FireQuery sÃ³ inicia se especialistas obrigatÃ³rios estiverem saudÃ¡veis
- [x] `fquery doctor` funciona
- [x] `fquery devices` funciona
- [x] contrato em inglÃªs Ã© obrigatÃ³rio
- [x] request invÃ¡lida Ã© rejeitada
- [x] write sem permissÃ£o Ã© rejeitado
- [x] fallback CPU funciona
- [x] MCP funciona
- [x] testes passam

---

## Parte 3 â€” Production Hardening

### Fase 30 â€” Storage Safety

- [x] Implementar compaction real
- [x] Criar backup de `.fbrain`
- [x] Criar restore de `.fbrain`
- [x] Validar integridade do brainfile na abertura
- [x] Definir estratÃ©gia de lock concorrente
- [x] Criar testes de corrupÃ§Ã£o e recovery

### Fase 31 â€” Compatibility

- [x] Definir polÃ­tica de `format_version`
- [x] Criar framework de migraÃ§Ã£o
- [x] Testar upgrade de versÃ£o do `.fbrain`
- [x] Definir comportamento para versÃ£o incompatÃ­vel
- [x] Documentar upgrade e downgrade

### Fase 32 â€” Observability

- [x] Criar logs estruturados
- [x] Definir cÃ³digos de erro estÃ¡veis
- [x] Padronizar traces de engine e pipeline
- [x] Criar diagnÃ³sticos legÃ­veis por mÃ¡quina

### Fase 33 â€” Reliability

- [x] Criar testes concorrentes de leitura e escrita
- [x] Criar testes de reopen repetido
- [x] Criar testes com arquivos grandes
- [x] Criar testes de failure injection
- [x] Criar benchmark bÃ¡sico das operaÃ§Ãµes principais

### Fase 34 â€” Release Readiness

- [x] Criar guia de deploy local
- [x] Criar guia de backup e restore
- [x] Criar guia de recovery
- [x] Definir checklist de `v0.1-beta`
- [x] Definir checklist de `v1.0`
