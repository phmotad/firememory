---
name: firememory-setup
description: Set up FireMemory for the current project. Checks installation, creates a .fbrain brainfile, configures the MCP server for the active editor, stores initial project context, and teaches the user how to use memory tools. Use when the user wants to start using FireMemory or asks how to set it up.
disable-model-invocation: true
allowed-tools: Bash Read
argument-hint: "[editor: cursor|claude-code|windsurf|zed]"
---

# FireMemory Setup

Running diagnostics before guiding setup.

## System state

```!
echo "=== FireMemory installation ==="
fmem version 2>/dev/null && echo "STATUS: installed" || echo "STATUS: not installed"
fquery version 2>/dev/null && echo "FQUERY: installed" || echo "FQUERY: not installed"
echo ""
echo "=== Current directory ==="
pwd
echo ""
echo "=== Existing .fbrain files ==="
ls *.fbrain .firememory/*.fbrain 2>/dev/null || echo "(none found)"
echo ""
echo "=== Requested editor ==="
echo "${ARGUMENTS:-not specified}"
```

---

## Instructions

Use the system state printed above to guide the user step by step.
Respond in the same language the user is speaking (PT-BR or English).

---

### STEP 1 — Install (skip if already installed)

If `STATUS: not installed` appears above, show the appropriate install command:

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.sh | bash
# Then restart the terminal.
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.ps1 | iex
```

**Homebrew:**
```sh
brew tap phmotad/firememory && brew install firememory
```

After installing, ask the user to restart the terminal and run `/firememory-setup` again.
If already installed, continue to Step 2.

---

### STEP 2 — Create a brainfile

If no `.fbrain` file was found above, create one for this project.
Suggest the name `<project-name>.fbrain` based on the current directory name:

```sh
fmem init ./<project-name>.fbrain
```

Or use the automatic default (created on first use):
```sh
fmem default   # shows where the default brainfile will live
```

Explain: the `.fbrain` file is the local memory database for this project.
It should be added to `.gitignore` (FireMemory does this automatically if you run `fmem init`).

---

### STEP 3 — Configure the editor MCP

Determine the editor:
- If `$ARGUMENTS` specifies an editor, use it.
- Otherwise, ask the user: "Which editor are you using? (cursor / claude-code / windsurf / zed)"

Then run:
```sh
fquery init-mcp <editor>
```

This writes the MCP server entry into the editor's config and prints the file it modified.

Ask the user to **fully restart** the editor (quit and reopen — not just reload).

If `fquery init-mcp` fails, show the manual JSON config and point to `docs/guides/<editor>-en.md` (or `<editor>-pt-BR.md`) for details:
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

---

### STEP 4 — Store initial project context

Ask the user: "Tell me about this project in 1–2 sentences. What does it do?"

Then store it as the first memory:
```sh
fmem remember ./<project-name>.fbrain "<user's description>"
```

Optionally ask for one more fact: tech stack, main language, or key convention.

Example:
```sh
fmem remember ./<project-name>.fbrain "Uses Go 1.24, PostgreSQL, and gRPC. REST API is deprecated."
```

---

### STEP 5 — Verify

Run both checks:
```sh
fmem stats ./<project-name>.fbrain   # should show at least 1 memory
fquery doctor                         # all checks should be green
```

If models are not downloaded yet:
```sh
fquery models pull    # downloads ~325 MB, runs once
```

---

### STEP 6 — Teach the user

Once setup is complete, explain the three things they'll actually use day-to-day:

**The agent uses memory automatically** — once FireQuery is connected via MCP, the editor's AI agent calls `remember`, `recall`, and `get_context` on its own. No manual steps needed.

**Manual recall (to inspect what's stored):**
```sh
fmem recall ./<project-name>.fbrain "your query"
```

**Manual context window (useful before a large task):**
```sh
fmem context ./<project-name>.fbrain "describe the current task"
```

**Add memory manually:**
```sh
fmem remember ./<project-name>.fbrain "important fact to preserve"
```

Point them to `docs/guides/` in the FireMemory repo, or online:
https://github.com/phmotad/firememory/tree/main/docs/guides

---

Setup complete. The project now has persistent semantic memory.
