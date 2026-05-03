# Compatibility

## Purpose

This document defines how FireMemory treats persisted `.fbrain` versions.

## Version Types

FireMemory tracks two different concepts:

- product version, such as `0.1.0`
- brainfile `format_version`, such as `0.1`

The compatibility contract is based on `format_version`.

## Current Policy

- current writable format: `0.1`
- readable legacy format: `0.0`
- unsupported future formats: rejected

## Upgrade

- opening a supported legacy Brainfile triggers migration to the current format
- migration is in-place
- migration updates the manifest and ensures current namespaces exist
- backup is strongly recommended before upgrade

## Downgrade

- downgrade is not automatic
- the supported downgrade mechanism is restore from a backup made before migration

## Operational Guidance

1. Run `fmem backup ./agent.fbrain ./agent.pre-upgrade.bak`
2. Open or use the brainfile with the current FireMemory binary
3. Verify with `fmem inspect ./agent.fbrain`
4. If rollback is needed, run `fmem restore ./agent.pre-upgrade.bak ./agent.fbrain`
