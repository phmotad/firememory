# Basic Example

This example shows the minimum end-to-end CLI flow for a local Brainfile.

## Commands

```sh
go run ./cmd/fmem init ./agent.fbrain

go run ./cmd/fmem remember ./agent.fbrain "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2"

go run ./cmd/fmem remember ./agent.fbrain "Joao relatou novamente problema fiscal em nota eletronica depois da versao 3.2"

go run ./cmd/fmem recall ./agent.fbrain "problema fiscal NF-e"

go run ./cmd/fmem sync ./agent.fbrain

go run ./cmd/fmem context ./agent.fbrain "responder Joao sobre erro fiscal apos atualizacao"
```

## Expected Outcome

- The Brainfile is created as a single `.fbrain` file.
- The first `remember` creates a new memory.
- The second `remember` creates or reinforces depending on the embedding and dedup path.
- `recall` returns ranked memories related to the fiscal issue.
- `sync` enriches pending memories with entities, facts, and relations.
- `context` returns a compact response-ready context block.

## Demo Script

PowerShell demo:

```powershell
./examples/basic/demo.ps1
```
