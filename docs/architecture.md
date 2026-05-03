# Architecture - FireMemory + FireQuery

## Overview

FireMemory is a local-first cognitive memory engine for AI agents.

It stores operational memory in a single `.fbrain` file.

FireQuery is a later interface layer that allows agents to interact with FireMemory through MCP and strict contracts.

## Development Order

FireMemory must be built first.

FireQuery must only start after FireMemory Core is functional.

## FireMemory Architecture

```txt
FireMemory
|- Brainfile Layer
|- Storage Layer
|- Memory Engine
|- Embedder Layer
|- Vector Engine
|- Graph Engine
|- Dedup Engine
|- Extractor Engine
|- Context Engine
|- CLI
`- MCP
```

## Brainfile

Official extension:

```txt
.fbrain
```

Example:

```txt
agent.fbrain
```

## Storage

The MVP uses bbolt internally.

bbolt is hidden behind the `Store` interface.

## FireQuery Architecture

```txt
Agent / LLM
    |
    v
   MCP
    |
    v
FireQuery
|- IntentClassifier
|- TriggerClassifier
|- EntityExtractor
|- FactExtractor
|- RelationClassifier
|- SimilarityEngine
|- Reranker
|- ContractBuilder
|- ContractValidator
|- DeviceManager
`- FireMemoryClient
    |
    v
FireMemory Core
    |
    v
agent.fbrain
```

## Core Rule

Models suggest.

Go validates.

FireMemory executes.

## FireQuery Contract

External agent communication can use user language.

Internal FireQuery -> FireMemory communication must be:

- English
- structured
- validated
- safe
- complete

If validation fails, FireQuery must reject the request.
