# Installing the firememory-setup skill

This skill guides you through setting up FireMemory in any project.

## Install globally (available in all your projects)

```sh
mkdir -p ~/.claude/skills/firememory-setup
cp SKILL.md ~/.claude/skills/firememory-setup/SKILL.md
```

Or clone the whole directory:

```sh
cp -r .claude/skills/firememory-setup ~/.claude/skills/
```

## Use it

Open any project in Claude Code and type:

```
/firememory-setup
```

Or with a specific editor:

```
/firememory-setup cursor
/firememory-setup claude-code
/firememory-setup windsurf
/firememory-setup zed
```

## What it does

1. Checks if FireMemory is installed — shows install commands if not
2. Creates a `.fbrain` brainfile for the current project
3. Runs `fquery init-mcp <editor>` to configure the MCP server
4. Asks you to describe the project and stores it as the first memory
5. Verifies everything is working
6. Teaches you the three commands you'll actually use
