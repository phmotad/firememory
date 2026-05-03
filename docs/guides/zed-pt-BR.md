# Zed — Guia de Configuração

> [English version](zed-en.md)

Este guia mostra como conectar o **Zed** ao FireMemory através do servidor MCP do FireQuery.

## Arquitetura

```
Zed  →  fquery mcp (stdio)  →  arquivo .fbrain
```

---

## Configuração rápida (binário instalado)

```sh
fquery init-mcp zed
```

Esse comando escreve a entrada do servidor MCP no `~/.config/zed/settings.json` do Zed e imprime o arquivo modificado. Reinicie o Zed e está pronto.

### Verificar

```sh
fquery init-mcp zed --print   # mostra o que foi escrito
fquery doctor                 # verifica status dos modelos e do ORT
```

---

## Configuração manual

Adicione ao `~/.config/zed/settings.json` dentro de `"context_servers"`:

```json
{
  "context_servers": {
    "firequery": {
      "command": {
        "path": "fquery",
        "args": ["mcp"]
      }
    }
  }
}
```

### Usar um brainfile específico por projeto

```json
{
  "context_servers": {
    "firequery": {
      "command": {
        "path": "fquery",
        "args": ["mcp"],
        "env": {
          "FIREMEMORY_DEFAULT_BRAIN": "/caminho/para/projeto.fbrain"
        }
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

**O Zed não mostra as ferramentas do FireQuery**
1. Reinicie o Zed.
2. Execute `fquery init-mcp zed --print` para verificar a configuração.
3. Execute `fquery doctor` — todas as verificações devem estar verdes.

**A primeira inicialização é lenta**
- Normal — os modelos (~325 MB) são baixados uma única vez na primeira execução.
