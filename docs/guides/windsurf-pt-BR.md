# Windsurf — Guia de Configuração

> [English version](windsurf-en.md)

Este guia mostra como conectar o **Windsurf** ao FireMemory através do servidor MCP do FireQuery.

## Arquitetura

```
Windsurf  →  fquery mcp (stdio)  →  arquivo .fbrain
```

---

## Configuração rápida (binário instalado)

```sh
fquery init-mcp windsurf
```

Esse comando escreve a entrada do servidor MCP na configuração do Windsurf e imprime o arquivo modificado. Reinicie o Windsurf e está pronto.

### Verificar

```sh
fquery init-mcp windsurf --print   # mostra o que foi escrito
fquery doctor                      # verifica status dos modelos e do ORT
```

---

## Configuração manual

O Windsurf armazena a configuração MCP em `~/.codeium/windsurf/mcp_config.json`:

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

**Windows** (`fquery` requer WSL2 ou Docker)
```json
{
  "mcpServers": {
    "firequery": {
      "command": "wsl",
      "args": ["fquery", "mcp"]
    }
  }
}
```

### Usar um brainfile específico por projeto

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

## Ferramentas disponíveis

| Ferramenta | O que faz |
|---|---|
| `firequery.remember` | Armazena uma memória durável |
| `firequery.recall` | Busca semântica |
| `firequery.get_context` | Janela de contexto ranqueada para uma tarefa |
| `firequery.explain` | Depura uma memória ou resultado de recuperação |
| `firequery.sync` | Executa enriquecimento de entidades/relações |

---

## Solução de problemas

**O Windsurf não mostra as ferramentas do FireQuery**
1. Reinicie o Windsurf completamente.
2. Execute `fquery init-mcp windsurf --print` para verificar a configuração.
3. Execute `fquery doctor` — todas as verificações devem estar verdes.

**A primeira inicialização é lenta**
- Normal — os modelos (~325 MB) são baixados uma única vez na primeira execução.
