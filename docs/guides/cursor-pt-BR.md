# Cursor — Guia de Configuração

> [English version](cursor-en.md)

Este guia mostra como conectar o **Cursor** ao FireMemory através do servidor MCP do FireQuery.

## Arquitetura

```
Cursor  →  fquery mcp (stdio)  →  arquivo .fbrain
```

O Cursor fala com o **FireQuery**, não diretamente com o FireMemory.

---

## Configuração rápida (binário instalado)

Se você instalou o FireMemory via script de instalação, Homebrew ou Scoop:

```sh
fquery init-mcp cursor
```

Esse comando escreve a entrada do servidor MCP na configuração do Cursor e imprime o arquivo modificado. Reinicie o Cursor e está pronto.

### Verificar

```sh
fquery init-mcp cursor --print   # mostra o que foi escrito
fquery doctor                    # verifica status dos modelos e do ORT
```

---

## Configuração manual

Se preferir configurar manualmente, adicione isso na configuração MCP do Cursor (normalmente `~/.cursor/mcp.json`):

**macOS / Linux**
```json
{
  "mcpServers": {
    "firequery": {
      "command": "fquery",
      "args": ["mcp"]
    }
  }
}
```

**Windows**
```json
{
  "mcpServers": {
    "firequery": {
      "command": "C:\\Users\\<voce>\\AppData\\Local\\firememory\\bin\\fquery.exe",
      "args": ["mcp"]
    }
  }
}
```

### Usar um brainfile específico por projeto

Adicione um bloco `env`:

```json
{
  "mcpServers": {
    "firequery": {
      "command": "fquery",
      "args": ["mcp"],
      "env": {
        "FIREMEMORY_DEFAULT_BRAIN": "/caminho/para/projeto.fbrain"
      }
    }
  }
}
```

---

## Configuração a partir do código-fonte

Se você está compilando a partir do código:

```sh
# compilar
make build

# criar um brainfile
./bin/fmem init ./agent.fbrain

# verificar saúde do FireQuery
./bin/fquery doctor
```

Depois adicione à configuração do Cursor:

```json
{
  "mcpServers": {
    "firequery": {
      "command": "/caminho/absoluto/para/bin/fquery",
      "args": ["mcp"],
      "cwd": "/caminho/absoluto/para/firememory"
    }
  }
}
```

---

## Ferramentas disponíveis

Após conectado, o Cursor pode usar:

| Ferramenta | O que faz |
|---|---|
| `firequery.remember` | Armazena uma memória durável |
| `firequery.recall` | Busca semântica |
| `firequery.get_context` | Janela de contexto ranqueada para uma tarefa |
| `firequery.explain` | Depura uma memória ou resultado de recuperação |
| `firequery.sync` | Executa enriquecimento de entidades/relações |

---

## Fluxo recomendado

**Antes de uma edição grande ou tarefa de raciocínio** — chame `firequery.get_context` com a descrição da tarefa atual.

**Após uma decisão importante** — chame `firequery.remember` para persistir.

**Quando um resultado de recall parecer errado** — chame `firequery.explain` para inspecionar.

---

## Solução de problemas

**O Cursor não mostra as ferramentas do FireQuery**
1. Reinicie o Cursor completamente (feche e reabra).
2. Execute `fquery init-mcp cursor --print` para verificar se a configuração foi escrita.
3. Execute `fquery doctor` — todas as verificações devem estar verdes.

**`fquery mcp` trava na inicialização**
- Execute `fquery models list` — os modelos podem estar faltando.
- Execute `fquery models pull` para baixá-los.

**A primeira inicialização é lenta**
- Normal — os modelos (~325 MB) são baixados uma única vez na primeira execução.

**O Cursor tenta falar diretamente com o FireMemory**
- Esse fluxo está errado. Configure o MCP apontando para `fquery`, não para `fmem`.
