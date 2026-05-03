# Backup and Restore

## Purpose

This document defines the operational backup and restore flow for `.fbrain`.

## When to Back Up

Create a backup:

- before migration
- before manual recovery steps
- before beta upgrades
- before destructive test scenarios

## Backup Command

```sh
fmem backup ./agent.fbrain ./backups/agent-2026-05-02.bak
```

JSON form:

```sh
fmem backup ./agent.fbrain ./backups/agent-2026-05-02.bak --json
```

## Restore Command

```sh
fmem restore ./backups/agent-2026-05-02.bak ./agent.fbrain
```

JSON form:

```sh
fmem restore ./backups/agent-2026-05-02.bak ./agent.fbrain --json
```

## Recommended Workflow

1. Stop the process using the target `.fbrain`.
2. Create a backup.
3. Perform the upgrade, migration, or test.
4. Validate with `fmem inspect`.
5. If rollback is needed, restore the backup.

## Validation After Restore

Run:

```sh
fmem inspect ./agent.fbrain
fmem recall ./agent.fbrain "sanity check"
```

Optional JSON validation:

```sh
fmem inspect ./agent.fbrain --json
```

## Operational Notes

- restore validates the resulting `.fbrain` before finalizing replacement
- backup does not mutate the source brainfile
- downgrade is performed through restore from a previous backup, not by reverse migration

## File Handling Rules

- keep backups outside the active working directory when possible
- use timestamped backup names
- do not overwrite your only known-good backup
- do not restore while another process holds the `.lock` for the same brainfile
