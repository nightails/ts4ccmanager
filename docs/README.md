# ts4ccmanager Documentation

Welcome to the documentation for **ts4ccmanager**, a fast, lightweight, and dependable mod and custom content (CC) manager for *The Sims 4* written in Go.

The goal of `ts4ccmanager` is to maximize player and developer efficiency by eliminating manual file juggling, preventing duplicate installations, safeguarding against archive path traversal, and tracking mod metadata in a local SQLite database with zero CGO dependencies.

---

## Documentation Index

### 1. [User Guide](user-guide.md)
A practical walkthrough for players and mod managers:
- Installing single packages (`.package`) and script mods (`.ts4script`).
- Installing multi-file mod archives (`.zip`).
- Listing installed mods with tabular and JSON formatting.
- Updating mods safely while preserving metadata.
- Removing mods cleanly without leaving orphaned files.
- Troubleshooting common installation issues.

### 2. [CLI Reference](cli-reference.md)
Detailed specification for all command-line interface commands:
- `ts4ccmanager add`
- `ts4ccmanager list`
- `ts4ccmanager update`
- `ts4ccmanager remove`
- `ts4ccmanager config` (`show`, `get`, `set`)
- Command flags, arguments, exit codes, and output formats.

### 3. [Configuration Guide](configuration.md)
How `ts4ccmanager` discovers paths and manages settings:
- Configuration file location (`config.json`) following XDG standards.
- Database location (`mods.db`).
- Automated Steam Proton / Linux path detection algorithm.
- Manual directory overrides and custom paths.

### 4. [System Architecture](architecture.md)
Engineering design and internal implementation details:
- Layered service architecture (CLI -> Service -> Repository / File Manager / Detector).
- SQLite relational schema, indexing, and data models.
- Integrity guarantees, rollback on failure, and duplicate detection.
- Security mechanisms: Zip Slip vulnerability prevention and path sanitization.

### 5. [Development & Contributing Guide](development.md)
Guidelines for building, testing, and extending `ts4ccmanager`:
- Development prerequisites and build steps.
- Running unit and integration test suites.
- In-memory database testing methodology.
- Code style and conventions for adding new features.

---

## Quick Start Summary

```bash
# 1. Inspect resolved configuration (automatically detects Steam Proton on Linux)
ts4ccmanager config show

# 2. Add custom content or a script mod
ts4ccmanager add ~/Downloads/curly_hair.package --author "SimArtist" --version "1.0"
ts4ccmanager add ~/Downloads/gameplay_overhaul.zip --name "Gameplay Overhaul"

# 3. View installed mods
ts4ccmanager list

# 4. Update an existing mod
ts4ccmanager update "Gameplay Overhaul" ~/Downloads/gameplay_overhaul_v2.zip --version "2.0"

# 5. Remove a mod
ts4ccmanager remove "curly_hair"
```
