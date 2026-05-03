# Codex Guide

This guide shows how to use FireMemory and FireQuery with Codex.

The supported flow is:

Codex -> FireQuery MCP -> FireMemory -> `agent.fbrain`

Codex should not connect to FireMemory directly.

## 1. Open the repository root

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

## 5. Seed durable memory

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

## 7. MCP command for Codex

The MCP command is:

```powershell
.\bin\fquery.exe mcp
```

If your Codex environment supports MCP command registration, point it to that command.

## 8. MCP registration example

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

## 9. Recommended Codex workflow

Before a task:

- call `firequery.get_context`
- or call `firequery.ask`

After an important stable fact or project decision:

- call `firequery.remember`

When you need to inspect retrieval behavior:

- call `firequery.explain`

## 10. Useful tool names

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## 11. Minimal setup summary

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
