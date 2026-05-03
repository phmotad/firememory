# Local Deploy Guide

## Purpose

This guide defines the minimum local deployment shape for FireMemory and FireQuery.

It is intended for internal beta and controlled technical users.

## Prerequisites

- Go `1.24.x`
- local filesystem with write access
- no external database
- no network dependency for the default test path

## Repository Validation

Before using the binaries:

```sh
go test ./...
```

## Build Commands

Build FireMemory CLI:

```sh
go build -o ./bin/fmem ./cmd/fmem
```

Build FireQuery CLI:

```sh
go build -o ./bin/fquery ./cmd/fquery
```

## Initial Brainfile Setup

Create a Brainfile:

```sh
./bin/fmem init ./agent.fbrain
```

Inspect it:

```sh
./bin/fmem inspect ./agent.fbrain
```

## Basic Operational Flow

Store memory:

```sh
./bin/fmem remember ./agent.fbrain "Client Joao uses Firebird 2.5"
```

Recall memory:

```sh
./bin/fmem recall ./agent.fbrain "firebird client"
```

Sync memory:

```sh
./bin/fmem sync ./agent.fbrain
```

Build context:

```sh
./bin/fmem context ./agent.fbrain "answer Joao about fiscal issue"
```

## FireQuery Runtime Check

Inspect devices:

```sh
./bin/fquery devices
```

Inspect readiness:

```sh
./bin/fquery doctor
```

JSON diagnostics:

```sh
./bin/fquery doctor --json
```

Start the MCP server:

```sh
./bin/fquery mcp
```

## Local Environment Notes

Optional backend flags:

- `FIREQUERY_ENABLE_CUDA=1`
- `FIREQUERY_ENABLE_DIRECTML=1`
- `FIREQUERY_ENABLE_COREML=1`
- `FIREQUERY_ENABLE_OPENVINO=1`

CPU fallback remains the safe default.

## Deployment Guardrails

- keep one active writer process per `.fbrain`
- treat `.fbrain` and its `.lock` sidecar as a unit while the process is running
- create a backup before migration or restore operations
- do not edit `.fbrain` manually

## Beta Recommendation

For internal beta:

- use a dedicated Brainfile per agent or workflow
- keep backups outside the working directory
- enable JSON diagnostics in automation and wrappers
