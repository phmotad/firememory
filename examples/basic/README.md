# Basic CLI Flow / Fluxo Básico da CLI

> [English](#english) | [Português](#português)

---

## English

This example shows the core `fmem` workflow: store memories, search them, and build a context window.

### Prerequisites

- FireMemory installed (`fmem version` should print a version number)

### 1. Create a brainfile

```sh
fmem init ~/demo.fbrain
```

### 2. Store memories

```sh
fmem remember ~/demo.fbrain "Client Joao uses Firebird 2.5 on Windows Server 2019"
fmem remember ~/demo.fbrain "Joao reported a fiscal NF-e error after update 3.2 on 2024-03-15"
fmem remember ~/demo.fbrain "The NF-e error was caused by an outdated SEFAZ certificate"
fmem remember ~/demo.fbrain "Fix: update the SEFAZ certificate bundle and restart the fiscal service"
```

### 3. Semantic search

```sh
fmem recall ~/demo.fbrain "fiscal NF-e error"
```

### 4. Build a context window

```sh
fmem context ~/demo.fbrain "how to fix Joao's fiscal problem"
```

### 5. Run entity enrichment

```sh
fmem sync ~/demo.fbrain
```

Extracts entities (people, systems, versions) and builds relations.

### 6. Inspect and backup

```sh
fmem stats ~/demo.fbrain            # memory counts
fmem inspect ~/demo.fbrain          # manifest
fmem backup ~/demo.fbrain ~/demo-backup-$(date +%Y%m%d).fbrain
```

---

## Português

Este exemplo mostra o fluxo principal do `fmem`: armazenar memórias, buscá-las e construir uma janela de contexto.

### Pré-requisitos

- FireMemory instalado (`fmem version` deve imprimir um número de versão)

### 1. Criar um brainfile

```sh
fmem init ~/demo.fbrain
```

### 2. Armazenar memórias

```sh
fmem remember ~/demo.fbrain "Cliente Joao usa Firebird 2.5 no Windows Server 2019"
fmem remember ~/demo.fbrain "Joao reportou erro fiscal na NF-e após atualização 3.2 em 15/03/2024"
fmem remember ~/demo.fbrain "O erro na NF-e foi causado por certificado SEFAZ desatualizado"
fmem remember ~/demo.fbrain "Correção: atualizar o pacote de certificados SEFAZ e reiniciar o serviço fiscal"
```

### 3. Busca semântica

```sh
fmem recall ~/demo.fbrain "erro fiscal NF-e"
```

### 4. Construir janela de contexto

```sh
fmem context ~/demo.fbrain "como resolver o problema fiscal do Joao"
```

### 5. Enriquecimento de entidades

```sh
fmem sync ~/demo.fbrain
```

Extrai entidades (pessoas, sistemas, versões) e constrói relações.

### 6. Inspecionar e fazer backup

```sh
fmem stats ~/demo.fbrain
fmem inspect ~/demo.fbrain
fmem backup ~/demo.fbrain ~/demo-backup-$(date +%Y%m%d).fbrain
```
