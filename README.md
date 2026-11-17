# ts4ccmanager

> **⚠️ Early Development Stage**: This project is in active early development. APIs, configuration formats, and the database schema are subject to breaking changes without notice. Use at your own risk, and be prepared to migrate your data when upgrading.

`ts4ccmanager` is a lightweight mod manager for the game *The Sims 4*. It's a simple tool running in the terminal, written in Go, for quickly adding, managing, and removing mods and custom content (CC) for the game.

## Features

- **Automated Mod Tracking**: Track installed packages and script mods in a local SQLite database.
- **Fast CLI Workflow**: Add, list, remove, update, and configure mods directly from the command line.
- **Archive & File Support**: Seamlessly install and extract `.package`, `.ts4script`, and `.zip` archives.
- **Steam / Proton Auto-Detection**: Automatically detects the Sims 4 Mods directory in Linux Steam Proton installations with manual configuration override support.
- **Pure Go & Zero CGO**: Built with a CGO-free pure Go SQLite driver (`modernc.org/sqlite`) for single-binary portability across Linux, macOS, and Windows.

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/README.md) directory:
- [User Guide](docs/user-guide.md) - Player walkthrough, managing packages, scripts, and archives.
- [CLI Reference](docs/cli-reference.md) - Full command syntax, flags, arguments, and exit codes.
- [Configuration Guide](docs/configuration.md) - Settings, XDG directory specifications, and Steam Proton auto-detection.
- [System Architecture](docs/architecture.md) - Component design, database schema, transactional safety, and security.
- [Development & Contributing](docs/development.md) - Build instructions, testing suite, and developer guidelines.

## Installation

```bash
go build -o tscc .
```

## CLI Commands

Launch the placeholder terminal interface with `tscc -t` (or `tscc --tui`).

### Configuration
Manage application settings such as the target Mods folder and database location:

```bash
# Show current configuration and resolved paths
tscc config show

# Get a configuration value
tscc config get mods_dir

# Set a custom Sims 4 Mods directory
tscc config set mods_dir "/path/to/Electronic Arts/The Sims 4/Mods"

# Set a custom database location
tscc config set db_path "/path/to/mods.db"
```

### Adding Mods
Install a mod file or archive into the Sims 4 Mods folder and register it in the database:

```bash
tscc add /path/to/mod.package
tscc add /path/to/script_mod.ts4script
tscc add /path/to/archive.zip --name "Custom Content Pack" --version "1.0" --author "CreatorName"
```

### Listing Mods
View all installed mods with metadata:

```bash
# Display formatted table
tscc list

# Output as JSON
tscc list --json
```

### Updating Mods
Replace an existing mod with a newer version while preserving metadata:

```bash
tscc update <id|name> /path/to/new_version.package --version "2.0"
```

### Removing Mods
Safely delete the mod file(s) from the disk and remove the record from the database:

```bash
tscc remove <id|name>
```

## Architecture & Database

`ts4ccmanager` stores metadata in a local SQLite database (`mods.db` stored in `~/.config/ts4ccmanager/mods.db` on Linux).

### Mod Record Schema
- **`id`**: Unique numeric identifier.
- **`name`**: Display name or identifier for the mod.
- **`file_name`**: Target file name in the Mods directory.
- **`file_path`**: Absolute target path in the Sims 4 Mods folder.
- **`installed_files`**: Paths owned by an archive install; ordinary single-file mods may omit this field.
- **`file_hash`**: SHA256 checksum of the installed file for duplicate detection and integrity tracking.
- **`version`**: Optional version string.
- **`author`**: Optional author / creator name.
- **`installed_at`**: Timestamp of installation.
- **`status`**: State of the mod (`active` or `disabled`).

## Planned Features & Roadmap

- [x] Steam Proton / Linux directory auto-detection
- [x] Configuration management (`config` command)
- [x] SQLite metadata storage with pure Go driver
- [x] Basic CLI commands (`add`, `list`, `remove`, `update`, `config`)
- [ ] Mod conflict and duplicate checksum detection
- [ ] Mod enable / disable toggling without file deletion