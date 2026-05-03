# Guia Rápido: FireMemory + FireQuery + Cursor

Este guia mostra, em português, o passo a passo para:

1. gerar o arquivo `.fbrain`
2. compilar o `FireQuery`
3. conectar o projeto ao Cursor via MCP

O fluxo correto é:

Cursor -> FireQuery MCP -> FireMemory -> `agent.fbrain`

O Cursor não deve falar com o FireMemory diretamente.

## 1. Pré-requisitos

No Windows, abra um terminal na raiz do projeto:

```powershell
cd C:\Projects\FireMemory
```

Você precisa ter:

- Go `1.24.x`
- permissão de escrita na pasta do projeto

## 2. Validar o projeto

Antes de usar, rode os testes:

```powershell
go test ./...
```

## 3. Gerar os binários

Compile o CLI do FireMemory:

```powershell
go build -o .\bin\fmem.exe .\cmd\fmem
```

Compile o CLI do FireQuery:

```powershell
go build -o .\bin\fquery.exe .\cmd\fquery
```

## 4. Gerar o banco `.fbrain`

O `.fbrain` é o arquivo de memória local do agente.

Crie o arquivo:

```powershell
.\bin\fmem.exe init .\agent.fbrain
```

Se quiser conferir:

```powershell
.\bin\fmem.exe inspect .\agent.fbrain
```

## 5. Colocar memória inicial no `.fbrain`

Exemplo:

```powershell
.\bin\fmem.exe remember .\agent.fbrain "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2"
.\bin\fmem.exe remember .\agent.fbrain "Joao relatou novamente problema fiscal na NF-e apos a versao 3.2"
```

Rodar o slow path:

```powershell
.\bin\fmem.exe sync .\agent.fbrain
```

Testar recuperação:

```powershell
.\bin\fmem.exe recall .\agent.fbrain "erro fiscal NF-e"
.\bin\fmem.exe context .\agent.fbrain "responder Joao sobre erro fiscal apos atualizacao"
```

## 6. Validar o FireQuery

Veja se o runtime está saudável:

```powershell
.\bin\fquery.exe doctor
```

Veja os devices detectados:

```powershell
.\bin\fquery.exe devices
```

## 7. Subir o MCP do FireQuery

O servidor MCP externo do projeto é:

```powershell
.\bin\fquery.exe mcp
```

Esse comando normalmente fica sendo executado pelo cliente MCP.

Você não precisa deixar ele aberto manualmente se o Cursor iniciar o processo para você.

## 8. Conectar ao Cursor

O Cursor deve registrar um servidor MCP por `stdio`.

Use esta definição como referência:

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

### Campos importantes

- `command`: caminho completo do `fquery.exe`
- `args`: deve conter `mcp`
- `cwd`: raiz do projeto FireMemory

## 9. O que pedir para o Cursor usar

Depois de conectado, o Cursor deve usar as tools do FireQuery:

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## 10. Fluxo recomendado no dia a dia

### Antes de uma tarefa importante

Peça contexto:

- use `firequery.get_context`
- ou `firequery.ask` com a tarefa atual

### Quando surgir uma informação durável

Peça persistência:

- use `firequery.remember`

### Quando quiser entender uma decisão da memória

Use:

- `firequery.explain`

## 11. Exemplo de request útil

Exemplo de payload para `firequery.get_context`:

```json
{
  "version": "0.1",
  "request_id": "req_context_001",
  "language": "pt-BR",
  "actor": {
    "type": "agent",
    "id": "cursor"
  },
  "brain": "./agent.fbrain",
  "input": {
    "task": "responder Joao sobre erro fiscal apos atualizacao",
    "budget_tokens": 1500,
    "include_graph": true,
    "include_trace": true
  }
}
```

## 12. Problemas comuns

### O Cursor não encontra o comando

Verifique se este arquivo existe:

```powershell
dir .\bin\fquery.exe
```

### O `.fbrain` ainda não existe

Crie com:

```powershell
.\bin\fmem.exe init .\agent.fbrain
```

### O Cursor conecta, mas não responde direito

Teste localmente primeiro:

```powershell
.\bin\fquery.exe doctor
.\bin\fmem.exe inspect .\agent.fbrain
```

### O agente tenta falar direto com o FireMemory

Esse fluxo está errado.

O correto é sempre:

Cursor -> FireQuery -> FireMemory

## 13. Resumo mínimo

Se quiser só o essencial:

```powershell
cd C:\Projects\FireMemory
go build -o .\bin\fmem.exe .\cmd\fmem
go build -o .\bin\fquery.exe .\cmd\fquery
.\bin\fmem.exe init .\agent.fbrain
.\bin\fquery.exe doctor
```

Depois registre no Cursor:

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
