# Recovery Guide

## Purpose

This guide defines what to do when a `.fbrain` cannot be opened or validated.

## Typical Failure Signals

Common signals:

- integrity validation failure
- unsupported `format_version`
- storage lock conflict
- missing manifest
- corrupted namespace layout

## First Response

1. Stop all processes using the target `.fbrain`.
2. Keep the original file untouched.
3. Copy the file to an investigation path.
4. Run `fmem inspect` against the original if possible.

## If the Brainfile Is Locked

Signal:

- `FMEM_STORAGE_LOCKED`

Action:

1. Identify the process still using the Brainfile.
2. Stop it cleanly.
3. Retry the command.

Do not delete the `.lock` sidecar while a real process is still active.

## If Integrity Validation Fails

Signal:

- `FMEM_INTEGRITY_VIOLATION`

Action:

1. Restore from the latest known-good backup.
2. Re-run `fmem inspect`.
3. Re-run a minimal `recall` sanity check.

## If the Format Version Is Unsupported

Signal:

- `FMEM_UNSUPPORTED_FORMAT_VERSION`

Action:

1. Verify the binary version being used.
2. Check the compatibility policy in `docs/operations/compatibility.md`.
3. Use the matching binary or restore a compatible backup.

## If the Manifest Is Missing

Signal:

- `FMEM_MANIFEST_NOT_FOUND`

Action:

1. Treat the file as corrupted.
2. Restore from backup.
3. Do not attempt manual reconstruction in production.

## Recovery Validation

After recovery:

```sh
fmem inspect ./agent.fbrain
fmem recall ./agent.fbrain "sanity check"
fquery doctor
```

JSON diagnostics:

```sh
fmem inspect ./agent.fbrain --json
fquery doctor --json
```

## Escalation Rule

If recovery requires manual bucket-level or raw file manipulation, the file is outside normal operational recovery and should be treated as forensic recovery, not standard support.
