# Cursor Guide

This guide shows how to:

1. create the `.fbrain` file
2. build `FireQuery`
3. connect Cursor to FireQuery through MCP

The supported flow is:

Cursor -> FireQuery MCP -> FireMemory -> `agent.fbrain`

Cursor should not talk to FireMemory directly.

## 1. Open the project root

On Windows:

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

## 5. Seed initial memory

```powershell
.\bin\fmem.exe remember .\agent.fbrain "Client Joao uses Firebird 2.5 and had a fiscal NF-e error after update 3.2"
.\bin\fmem.exe remember .\agent.fbrain "Joao reported the fiscal NF-e problem again after version 3.2"
.\bin\fmem.exe sync .\agent.fbrain
```

Optional recall check:

```powershell
.\bin\fmem.exe recall .\agent.fbrain "fiscal NF-e error"
.\bin\fmem.exe context .\agent.fbrain "answer Joao about the fiscal issue after the update"
```

## 6. Validate FireQuery

```powershell
.\bin\fquery.exe doctor
.\bin\fquery.exe devices
```

## 7. MCP command for Cursor

The MCP command is:

```powershell
.\bin\fquery.exe mcp
```

Cursor should launch that command as a `stdio` MCP server.

## 8. Cursor MCP registration

Use this command definition as the reference:

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

Important fields:

- `command`: full path to `fquery.exe`
- `args`: must include `mcp`
- `cwd`: repository root

## 9. Recommended tools in Cursor

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## 10. Recommended workflow

Before a large edit:

- use `firequery.get_context`
- or use `firequery.ask` with the current task

After an important durable decision:

- use `firequery.remember`

When debugging memory behavior:

- use `firequery.explain`

## 11. Minimal setup summary

```powershell
cd C:\Projects\FireMemory
go build -o .\bin\fmem.exe .\cmd\fmem
go build -o .\bin\fquery.exe .\cmd\fquery
.\bin\fmem.exe init .\agent.fbrain
.\bin\fquery.exe doctor
```

Then register this MCP server in Cursor:

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
