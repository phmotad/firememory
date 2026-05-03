# Claude Code Guide

This guide shows how to use FireMemory and FireQuery with Claude Code.

The supported architecture is:

Claude Code -> FireQuery MCP -> FireMemory -> `agent.fbrain`

Claude Code should not connect to FireMemory directly.

## 1. Go to the project root

```powershell
cd C:\Projects\FireMemory
```

## 2. Validate the repository

```powershell
go test ./...
```

## 3. Build the binaries

```powershell
go build -o .\bin\fmem.exe .\cmd\fmem
go build -o .\bin\fquery.exe .\cmd\fquery
```

## 4. Create the Brainfile

```powershell
.\bin\fmem.exe init .\agent.fbrain
```

Optional check:

```powershell
.\bin\fmem.exe inspect .\agent.fbrain
```

## 5. Add initial memory

```powershell
.\bin\fmem.exe remember .\agent.fbrain "Client Joao uses Firebird 2.5 and had a fiscal NF-e error after update 3.2"
.\bin\fmem.exe remember .\agent.fbrain "Joao reported the fiscal NF-e problem again after version 3.2"
.\bin\fmem.exe sync .\agent.fbrain
```

## 6. Validate FireQuery

```powershell
.\bin\fquery.exe doctor
.\bin\fquery.exe devices
```

## 7. MCP command for Claude Code

Use this command:

```powershell
.\bin\fquery.exe mcp
```

Claude Code should run it as a `stdio` MCP server.

## 8. MCP registration example

Use this command definition as the base:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```

## 9. Repository instructions for Claude Code

Point Claude Code to:

- [AGENTS.md](AGENTS.md)

That file is the canonical project instruction file.

## 10. Recommended Claude Code workflow

Use `firequery.get_context` before:

- long reasoning
- large code changes
- user-facing answer generation

Use `firequery.remember` for:

- durable user facts
- project decisions
- environment notes worth persisting

Use `firequery.explain` when:

- a recall result looks wrong
- a context result needs debugging

## 11. Useful tool names

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## 12. Minimal setup summary

```powershell
cd C:\Projects\FireMemory
go build -o .\bin\fmem.exe .\cmd\fmem
go build -o .\bin\fquery.exe .\cmd\fquery
.\bin\fmem.exe init .\agent.fbrain
.\bin\fquery.exe doctor
```

Then register:

```json
{
  "name": "firequery",
  "transport": {
    "type": "stdio",
    "command": "C:\\Projects\\FireMemory\\bin\\fquery.exe",
    "args": ["mcp"],
    "cwd": "C:\\Projects\\FireMemory"
  }
}
```
