# Claude Code — Guia de Configuração

> [English version](claude-code-en.md)

Este guia mostra como conectar o **Claude Code** ao FireMemory através do servidor MCP do FireQuery.

## Arquitetura

```
Claude Code  →  fquery mcp (stdio)  →  arquivo .fbrain
```

O Claude Code fala com o **FireQuery**, não diretamente com o FireMemory.

---

## Configuração rápida (binário instalado)

Se você instalou o FireMemory via script de instalação, Homebrew ou Scoop:

```sh
fquery init-mcp claude-code
```

Esse comando escreve a entrada do servidor MCP na configuração do Claude Code (`~/.claude/settings.json`) e imprime o arquivo modificado. Reinicie o Claude Code e está pronto.

### Verificar

```sh
fquery init-mcp claude-code --print   # mostra o que foi escrito
fquery doctor                         # verifica status dos modelos e do ORT
```

---

## Configuração manual

Adicione ao `~/.claude/settings.json`:

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

**Windows** (atenção: `fquery` requer WSL2 ou Docker no Windows)
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

Depois adicione ao `~/.claude/settings.json`:

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

## Integração com AGENTS.md

Aponte o Claude Code para o `AGENTS.md` do projeto para as instruções canônicas. O Claude Code lê esse arquivo automaticamente quando presente na raiz do repositório.

---

## Ferramentas disponíveis

Após conectado, o Claude Code pode usar:

| Ferramenta | O que faz |
|---|---|
| `firequery.remember` | Armazena uma memória durável |
| `firequery.recall` | Busca semântica |
| `firequery.get_context` | Janela de contexto ranqueada para uma tarefa |
| `firequery.explain` | Depura uma memória ou resultado de recuperação |
| `firequery.sync` | Executa enriquecimento de entidades/relações |

---

## Fluxo recomendado

**Antes de uma tarefa de raciocínio complexa ou mudança grande de código** — chame `firequery.get_context` com a tarefa atual.

**Após uma decisão durável ou fato do usuário** — chame `firequery.remember`.

**Quando um resultado de recall parecer errado** — chame `firequery.explain`.

---

## Solução de problemas

**O Claude Code não mostra as ferramentas do FireQuery**
1. Reinicie o Claude Code.
2. Execute `fquery init-mcp claude-code --print` para verificar a configuração.
3. Execute `fquery doctor` — todas as verificações devem estar verdes.

**`fquery mcp` trava na inicialização**
- Execute `fquery models pull` para baixar os modelos faltantes.

**A primeira inicialização é lenta**
- Normal — os modelos (~325 MB) são baixados uma única vez na primeira execução.
