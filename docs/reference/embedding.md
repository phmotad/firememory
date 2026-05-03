# Embeddings

## Overview

FireMemory uses a pluggable `Embedder` interface.

The MVP must work locally and offline in tests, so the default testing implementation is deterministic.

## Implementations

- `DeterministicEmbedder`
- `ExternalEmbedder`
- `E5Embedder`

## DeterministicEmbedder

`DeterministicEmbedder` is the default embedder for tests.

It produces stable vectors for the same input text and does not require network access or external models.

## ExternalEmbedder

`ExternalEmbedder` is the generic adapter for any external embedding provider.

It must:

- validate configured dimension
- validate returned vector dimension
- apply L2 normalization before returning the vector

## E5Embedder

`E5Embedder` is the adapter for:

```txt
intfloat/multilingual-e5-small
```

For compatibility with the E5 family, the adapter supports the standard text prefixes:

- `query:`
- `passage:`

The default conceptual dimension for the MVP is `384`.

## Normalization

All embedders must return L2-normalized vectors.

This keeps cosine similarity stable across implementations.

## Validation Rules

Embedder implementations must reject:

- empty input text
- invalid configured dimension
- returned vectors with the wrong dimension
- zero-magnitude vectors before normalization
