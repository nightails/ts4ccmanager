# ts4ccmanager CLI Reference

Complete reference for all commands, arguments, flags, and outputs supported by `ts4ccmanager`.

---

## Command Hierarchy

```text
ts4ccmanager [command]
├── -t, --tui
├── add <file-path>
├── list [--json]
├── update <id|name> <new-file-path>
├── remove <id|name>
└── config
    ├── show
    ├── get <key>
    └── set <key> <value>
```

---

## Global Flags & Environment

- `-h, --help`: Displays help text for any command or subcommand.
- `-t, --tui`: Launches the placeholder terminal user interface (for example, `tscc -t`).
- `XDG_CONFIG_HOME`: If defined on Unix systems, `ts4ccmanager` stores its configuration under `$XDG_CONFIG_HOME/ts4ccmanager/`. Defaults to `~/.config/ts4ccmanager/`.

---

## Commands

### `ts4ccmanager add`

Validates, copies, or extracts a mod file into the Sims 4 Mods directory and records it in the database.

#### Syntax
```bash
ts4ccmanager add <file-path> [flags]
```

#### Arguments
- `<file-path>` *(required)*: Path to the `.package`, `.ts4script`, or `.zip` file on disk.

#### Flags
| Flag | Shorthand | Type | Description |
| :--- | :--- | :--- | :--- |
| `--name` | `-n` | `string` | Custom display name for the mod (defaults to file base name without extension). |
| `--version` | `-v` | `string` | Mod version identifier (e.g. `1.0.2`, `v3`). |
| `--author` | `-a` | `string` | Mod creator or artist name. |

#### Example
```bash
ts4ccmanager add ~/Downloads/modern_couch.package -n "Modern Couch" -a "Peacemaker" -v "1.0"
```

#### Output
```text
Successfully added mod "Modern Couch" (ID: 1)
  File:        modern_couch.package
  Destination: /path/to/Mods/modern_couch.package
  Version:     1.0
  Author:      Peacemaker
```

#### Error Conditions
- `source mod file does not exist`: The supplied `<file-path>` is not found.
- `unsupported file format`: The file extension is not `.package`, `.ts4script`, or `.zip`.
- `mod is already installed (duplicate checksum found)`: An identical file (by SHA256) is already recorded.

---

### `ts4ccmanager list`

Displays all mods and custom content currently registered in the database.

#### Syntax
```bash
ts4ccmanager list [flags]
```

#### Flags
| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `--json` | `bool` | `false` | Formats output as an indented JSON array. |

#### Tabular Example
```bash
ts4ccmanager list
```
```text
ID   NAME           FILE NAME            VERSION   AUTHOR       STATUS   INSTALLED AT
1    Modern Couch   modern_couch.package 1.0       Peacemaker   active   2026-10-29 14:00:10
2    Skin Blend     skin_overlay.package 2.0       Lumia        active   2026-10-29 14:05:22
```

#### JSON Example
```bash
ts4ccmanager list --json
```
```json
[
  {
    "id": 1,
    "name": "Modern Couch",
    "file_name": "modern_couch.package",
    "file_path": "/path/to/Mods/modern_couch.package",
    "file_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "version": "1.0",
    "author": "Peacemaker",
    "installed_at": "2026-10-29T14:00:10Z",
    "status": "active"
  }
]
```

---

### `ts4ccmanager update`

Replaces an existing mod's file(s) on disk, computes new checksums, and updates metadata in the database.

#### Syntax
```bash
ts4ccmanager update <id|name> <new-file-path> [flags]
```

#### Arguments
- `<id|name>` *(required)*: The numeric database ID or exact name of the mod to update.
- `<new-file-path>` *(required)*: Path to the new `.package`, `.ts4script`, or `.zip` replacement file.

#### Flags
| Flag | Shorthand | Type | Description |
| :--- | :--- | :--- | :--- |
| `--name` | `-n` | `string` | Update the display name. |
| `--version` | `-v` | `string` | Update the version string. |
| `--author` | `-a` | `string` | Update the author string. |

#### Example
```bash
ts4ccmanager update "Modern Couch" ~/Downloads/modern_couch_v2.package --version "2.0"
```

#### Output
```text
Successfully updated mod "Modern Couch" (ID: 1)
  New File:    modern_couch_v2.package
  Destination: /path/to/Mods/modern_couch_v2.package
  Version:     2.0
  Author:      Peacemaker
```

---

### `ts4ccmanager remove`

Deletes the mod file or directory from the Sims 4 Mods directory and deletes the record from the database.

#### Syntax
```bash
ts4ccmanager remove <id|name>
```

#### Arguments
- `<id|name>` *(required)*: Numeric database ID or exact name of the mod.

#### Example
```bash
ts4ccmanager remove 1
# or
ts4ccmanager remove "Modern Couch"
```

#### Output
```text
Successfully removed mod "Modern Couch"
```

---

### `ts4ccmanager config`

Manage application configuration options.

#### Subcommands

#### 1. `ts4ccmanager config show`
Displays active configuration file path and resolved directory targets.
```bash
ts4ccmanager config show
```
```text
CONFIG FILE:                    /home/user/.config/ts4ccmanager/config.json
MODS DIRECTORY (CONFIGURED):    
MODS DIRECTORY (RESOLVED):      /home/user/.steam/steam/steamapps/compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4/Mods
DB PATH (CONFIGURED):           
DB PATH (RESOLVED):             /home/user/.config/ts4ccmanager/mods.db
```

#### 2. `ts4ccmanager config get <key>`
Retrieves the raw value of a configuration key.
```bash
ts4ccmanager config get mods_dir
```

#### 3. `ts4ccmanager config set <key> <value>`
Updates a configuration setting and persists it to `config.json`.
```bash
ts4ccmanager config set mods_dir "/path/to/Mods"
ts4ccmanager config set db_path "/path/to/custom_mods.db"
```

#### Valid Configuration Keys
- `mods_dir`: Absolute path to the target Sims 4 `Mods` directory.
- `db_path`: Absolute path to the SQLite `mods.db` database file.

---

## Exit Codes

| Exit Code | Meaning |
| :--- | :--- |
| `0` | Command executed successfully. |
| `1` | Command failed (e.g. invalid arguments, missing file, duplicate checksum, or database error). |
