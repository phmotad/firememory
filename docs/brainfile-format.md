# Brainfile Format

## Official Extension

FireMemory uses:

```txt
.fbrain
```

Examples:

```txt
agent.fbrain
support-agent.fbrain
company-memory.fbrain
```

## Why `.fbrain`

`.fbrain` means FireMemory Brainfile.

It avoids ambiguity with existing `.brain` usages and is more specific to the FireMemory ecosystem.

## MVP Implementation

The `.fbrain` file uses bbolt internally.

This is an implementation detail.

The public abstraction is Brainfile.

## Internal Namespaces

- manifest
- memories
- entities
- relations
- facts
- events
- concepts
- sources
- vectors
- hash_index
- graph_nodes
- graph_edges
- traces
- sync_queue

## Manifest

```json
{
  "id": "brain_xxx",
  "name": "agent",
  "version": "0.1.0",
  "format_version": "0.1",
  "extension": ".fbrain",
  "embedding_model": "deterministic",
  "embedding_dim": 384,
  "created_at": "...",
  "updated_at": "..."
}
```

## Compatibility

Every `.fbrain` must have:

- format version
- manifest
- embedding model metadata
- migration compatibility

## Format Version Policy

- `format_version` identifies the persisted Brainfile layout, not the product release version.
- FireMemory may open the current format version directly.
- FireMemory may auto-migrate older known format versions.
- FireMemory must reject unknown or newer unsupported format versions.

Current supported versions:

- `0.1` as the current format
- `0.0` as a legacy format that is auto-migrated to `0.1`

## Upgrade Policy

- Upgrades from supported older formats are in-place and automatic on open.
- Before an upgrade, users should create a backup of the `.fbrain`.
- After a successful migration, `format_version` is rewritten to the current value.

## Downgrade Policy

- Downgrade is not automatic.
- Once a Brainfile is migrated to a newer `format_version`, returning to an older format requires restore from backup.
- FireMemory does not attempt lossy downgrade transformations in the MVP or hardening phase.

## Unsupported Versions

- Unknown versions are rejected.
- Newer incompatible versions are rejected.
- Rejection is explicit and should trigger backup/upgrade review rather than silent fallback.
