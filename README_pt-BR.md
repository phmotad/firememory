# FireMemory

[![Test](https://github.com/phmotad/firememory/actions/workflows/test.yml/badge.svg)](https://github.com/phmotad/firememory/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/phmotad/firememory)](https://github.com/phmotad/firememory/releases/latest)
[![License](https://img.shields.io/github/license/phmotad/firememory)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/phmotad/firememory)](https://goreportcard.com/report/github.com/phmotad/firememory)

**Motor de memória semântica local para agentes de IA.**

> [English version](README.md)

O FireMemory armazena tudo em um único arquivo `.fbrain` — sem servidor, sem nuvem, sem configuração.
Agentes leem e escrevem memória via [MCP](https://modelcontextprotocol.io/) usando `fquery mcp`.
Os modelos de ML (~325 MB) são baixados automaticamente no primeiro uso.

---

## Início rápido (60 segundos)

### 1. Instalar

**macOS / Linux**
```sh
curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.sh | bash
```

**Windows** (PowerShell)
```powershell
irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.ps1 | iex
```

**Homebrew**
```sh
brew tap phmotad/firememory
brew install firememory
```

**Scoop**
```powershell
scoop bucket add phmotad https://github.com/phmotad/scoop-firememory
scoop install firememory
```

### 2. Configurar o editor

```sh
fquery init-mcp claude-code   # Claude Code
fquery init-mcp cursor        # Cursor
fquery init-mcp windsurf      # Windsurf
fquery init-mcp zed           # Zed
```

Esse comando escreve a entrada do servidor MCP no arquivo de configuração do editor e imprime o caminho modificado.

### 3. Criar um brainfile

```sh
fmem init ~/meu.fbrain
```

Ou pule essa etapa — `fmem stats` e qualquer chamada de ferramenta `fquery` criarão automaticamente `~/.firememory/default.fbrain` se não existir.

### 4. Reiniciar o editor

O servidor MCP inicia sob demanda. Na primeira chamada, `fquery mcp` baixa os três modelos de ML (~325 MB, executa uma única vez). Inicializações posteriores são instantâneas.

---

## O que é

FireMemory **não** é um banco de dados vetorial, **não** é uma camada RAG e **não** é SQL.

É um *motor de memória cognitiva*: entende o que está sendo armazenado, faz deduplicação semântica, constrói um grafo de conhecimento e monta janelas de contexto adaptadas a uma consulta.

| Conceito | FireMemory |
|---|---|
| Formato de armazenamento | Arquivo único `.fbrain` (bbolt) |
| Embeddings | multilingual-e5-small INT8 (ONNX local) |
| Extração de entidades | GLiNER-small-v2.1 INT8 (ONNX local) |
| Intent / classificação | DeBERTa-v3-small INT8 (ONNX local) |
| Tamanho dos modelos | ~325 MB total, baixados uma única vez |
| Transporte | MCP via stdio (`fquery mcp`) |
| Privacidade | 100% local — nada sai da sua máquina |

---

## Conectividade do agente

Agentes se comunicam com o **FireQuery** (a camada MCP), não diretamente com o FireMemory.

```
Seu agente de editor
      │  MCP (stdio)
      ▼
  fquery mcp          ← FireQuery: valida, classifica, enriquece
      │
      ▼
  arquivo .fbrain     ← FireMemory: armazena, recupera, sincroniza
```

### Ferramentas MCP disponíveis

| Ferramenta | Descrição |
|---|---|
| `remember` | Armazena uma memória (deduplicação automática) |
| `recall` | Busca semântica sobre memórias armazenadas |
| `get_context` | Recupera uma janela de contexto ranqueada para uma consulta |
| `sync` | Executa enriquecimento lento (entidades, relações, grafo) |
| `explain` | Explica uma memória armazenada |

---

## Referência da CLI

### fmem

```
fmem init <arquivo.fbrain>                  cria um novo brainfile
fmem remember <arquivo.fbrain> <texto>      armazena uma memória
fmem recall <arquivo.fbrain> <consulta>     busca semântica
fmem sync <arquivo.fbrain>                  enriquecimento de entidades/relações
fmem context <arquivo.fbrain> <consulta>    constrói uma janela de contexto
fmem inspect <arquivo.fbrain>               exibe o manifesto
fmem snapshot <arquivo.fbrain>              dump completo dos dados (JSON)
fmem backup <arquivo.fbrain> <destino>      copia para caminho de backup
fmem restore <backup> <arquivo.fbrain>      restaura do backup
fmem compact <arquivo.fbrain>               recupera espaço (vacuum do bbolt)
fmem stats [<arquivo.fbrain>]               contagem de memórias
fmem default                                imprime/cria o caminho padrão do brainfile
fmem version                                imprime a versão
```

### fquery

```
fquery mcp                              inicia o servidor MCP (stdio)
fquery init-mcp <cliente>              configura a entrada MCP do editor
  clientes: claude-code, cursor, windsurf, zed
  --print                               dry-run: exibe a configuração que seria escrita
  --config <caminho>                    substitui o caminho do arquivo de configuração
fquery models list                      exibe o status dos modelos baixados
fquery models pull                      baixa modelos faltantes
fquery models pull --force              rebaixa todos os modelos
fquery models gc                        remove modelos em cache
fquery devices                          lista dispositivos de computação (CPU/GPU)
fquery doctor                           executa diagnósticos
fquery version                          imprime a versão
```

---

## Modelos

O FireQuery usa três modelos ONNX INT8 locais, baixados automaticamente:

| Modelo | Uso | Tamanho |
|---|---|---|
| `multilingual-e5-small` | Embeddings, recall semântico | ~120 MB |
| `deberta-v3-small` | Classificação de intent e trigger | ~72 MB |
| `gliner-small-v2.1` | Extração de entidades nomeadas | ~121 MB |

Os modelos são armazenados em:
- **macOS** — `~/Library/Caches/firememory/models`
- **Linux** — `~/.cache/firememory/models`
- **Windows** — `%LOCALAPPDATA%\firememory\models`

Substitua com `FIREMEMORY_MODELS_DIR`.

Para remover: `fquery models gc`

---

## Docker

```sh
docker run --rm -i \
  -v "$HOME/.firememory/models:/models" \
  ghcr.io/phmotad/firequery mcp
```

Os modelos são armazenados no volume montado e baixados na primeira execução.

---

## Compilar a partir do código-fonte

Requer Go 1.24 e um compilador C (para CGO).

```sh
git clone https://github.com/phmotad/firememory
cd firememory
make build          # produz bin/fmem e bin/fquery (com -tags onnx)
make test           # executa todos os testes (offline-safe, sem modelos)
```

Os binários de release são construídos com runners nativos por plataforma e a biblioteca compartilhada do ONNX Runtime é incluída em cada arquivo (sem instalação separada).

---

## Arquitetura

```
cmd/fmem       — CLI do FireMemory
cmd/fquery     — CLI do FireQuery + servidor MCP

internal/
  engine/        — remember / recall / sync / context / explain
  storage/       — store bbolt por trás da interface Store
  brainfile/     — formato .fbrain, validação, migração
  dedup/         — deduplicação semântica (hash + embedding)
  embedder/      — interface Embedder (E5, determinístico, externo)
  graph/         — grafo de conhecimento (entidades + relações)
  firequery/     — camada de interface cognitiva (pipeline, MCP, contratos)
  firequery/onnx — backend de inferência ONNX (build tag: onnx)
  modelcache/    — download automático, verificação, extração de modelos ML
  initcfg/       — escreve entradas MCP nos arquivos de configuração do editor
  defaultbrain/  — caminho padrão do brainfile + auto-init
  version/       — string de versão injetada em tempo de build
```

Caminho rápido (`remember`): hash → embed → dedup → persist
Caminho lento (`sync`): extrair entidades → construir relações → atualizar grafo

---

## Agent Skill

Instale a skill `firememory-setup` para que seu agente de IA configure o FireMemory automaticamente em qualquer projeto.

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install-skill.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install-skill.ps1 | iex
```

Depois, dentro de qualquer projeto no Claude Code (ou outro [agente compatível](https://agentskills.io)):

```
/firememory-setup
```

A skill verifica a instalação do FireMemory, cria um `.fbrain`, conecta o servidor MCP ao seu editor e ensina os três comandos que você vai usar no dia a dia.

---

## Contribuindo

Veja [CONTRIBUTING.md](CONTRIBUTING.md). Todos os testes devem passar (`go test ./...`) antes de submeter um PR.

O backend ONNX está atrás de `//go:build onnx` — os testes rodam offline sem modelos por design.
