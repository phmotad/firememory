# MCP Client Registration / Registro de Clientes MCP

> [English](#english) | [Português](#português)

---

## English

This example shows how to register FireQuery as an MCP server in various editors.

**Architecture:**
```
Your editor  →  fquery mcp (stdio)  →  .fbrain file
```

Always connect to **FireQuery**, not directly to FireMemory.

### Quick registration (recommended)

```sh
fquery init-mcp cursor        # Cursor
fquery init-mcp claude-code   # Claude Code
fquery init-mcp windsurf      # Windsurf
fquery init-mcp zed           # Zed
```

Restart the editor after running the command.

### Manual registration

For editors that support the standard MCP JSON config:

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

**Windows** (fquery requires WSL2 or Docker)
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

### Per-project brainfile

```json
{
  "mcpServers": {
    "firequery": {
      "command": "fquery",
      "args": ["mcp"],
      "env": {
        "FIREMEMORY_DEFAULT_BRAIN": "/path/to/project.fbrain"
      }
    }
  }
}
```

See [docs/guides/](../../docs/guides/) for editor-specific guides.

---

## Português

Este exemplo mostra como registrar o FireQuery como servidor MCP em vários editores.

**Arquitetura:**
```
Seu editor  →  fquery mcp (stdio)  →  arquivo .fbrain
```

Sempre conecte ao **FireQuery**, não diretamente ao FireMemory.

### Registro rápido (recomendado)

```sh
fquery init-mcp cursor        # Cursor
fquery init-mcp claude-code   # Claude Code
fquery init-mcp windsurf      # Windsurf
fquery init-mcp zed           # Zed
```

Reinicie o editor após executar o comando.

### Registro manual

Para editores que suportam o formato JSON padrão do MCP:

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

**Windows** (fquery requer WSL2 ou Docker)
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

### Brainfile por projeto

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

Veja [docs/guides/](../../docs/guides/) para guias específicos por editor.
