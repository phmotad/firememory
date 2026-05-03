# Domain

## Core Types

FireMemory currently models:

- `Memory`
- `Entity`
- `Fact`
- `Relation`
- `Event`
- `Concept`
- `SourceRef`

## Memory

`Memory` is the central unit stored in the Brainfile.

Important fields:

- `id`
- `content`
- `normalized_content`
- `hash`
- `kind`
- `status`
- `scope`
- `importance`
- `confidence`
- `embedding_model`
- `embedding_dim`

## Memory Kinds

- `note`
- `fact`
- `event`
- `concept`

## Memory Status

- `active`
- `pending_sync`
- `synced`
- `forgotten`

## Relations

Current relation types:

- `references`
- `duplicate`
- `reinforce`
- `complement`
- `update`
- `conflict`
- `associated`

## Engine Contracts

The engine already exposes inputs and outputs for:

- `remember`
- `recall`
- `relate`
- `forget`
- `consolidate`
- `sync`
- `get_context`
- `explain`
