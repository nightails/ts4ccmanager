# ts4ccmanager Configuration Guide

This document details the configuration system, configuration keys, directory precedence, and the Steam Proton auto-detection algorithm in `ts4ccmanager`.

---

## Configuration Architecture

`ts4ccmanager` follows the **XDG Base Directory Specification** on Linux and Unix environments to store persistent settings and database files cleanly without polluting the user's home directory.

### Storage Locations

| Item | Default Location | Description |
| :--- | :--- | :--- |
| **Config Directory** | `$XDG_CONFIG_HOME/ts4ccmanager/` (or `~/.config/ts4ccmanager/`) | Base directory for application data. |
| **Config File** | `~/.config/ts4ccmanager/config.json` | JSON configuration file for user overrides. |
| **Database File** | `~/.config/ts4ccmanager/mods.db` | Local SQLite database containing mod records. |

---

## Configuration Keys

The `config.json` schema supports two primary settings:

```json
{
  "mods_dir": "",
  "db_path": ""
}
```

### `mods_dir`
- **Type**: `string` (path)
- **Default**: `""` (Empty string; triggers automatic Steam Proton detection)
- **Description**: The absolute path to the target *The Sims 4* `Mods` directory on your system.
- **Validation**: When set, `ts4ccmanager` validates that the directory exists on disk when resolving paths.

### `db_path`
- **Type**: `string` (path)
- **Default**: `""` (Empty string; defaults to `~/.config/ts4ccmanager/mods.db`)
- **Description**: The absolute path to the SQLite database file used for tracking mods.
- **Behavior**: If the parent directory does not exist, `ts4ccmanager` automatically creates it (`0755` permissions) upon connection.

---

## Directory Resolution & Precedence

### Resolution Flow for `mods_dir`

```text
+-----------------------------------+
| Is `mods_dir` explicitly set?     |
+-----------------+-----------------+
                  |
         +--------+--------+
         |                 |
       [YES]              [NO]
         |                 |
         v                 v
+-----------------+   +------------------------------------+
| Verify path     |   | Run Steam Proton Auto-Detection    |
| exists on disk  |   +-----------------+------------------+
+--------+--------+                     |
         |                     +--------+--------+
    +----+----+                |                 |
    |         |             [FOUND]          [NOT FOUND]
 [EXISTS]  [MISSING]           |                 |
    |         |                v                 v
    v         v           +----------+      +---------------------------+
  [USE]    [ERROR]        | Use Path |      | Return configuration      |
                          +----------+      | error with manual fix cmd |
                                            +---------------------------+
```

1. **Explicit Setting**: If `mods_dir` is configured in `config.json`, `ts4ccmanager` verifies the path with `os.Stat`. If the directory does not exist, an error is returned.
2. **Fallback Auto-Detection**: If `mods_dir` is not configured (`""`), the Steam Proton detection engine is invoked.
3. **Error Reporting**: If detection fails, the tool guides the user to set the path manually using `tscc config set mods_dir <path>`.

### Resolution Flow for `db_path`

1. If `db_path` is configured in `config.json`, that specific path is used.
2. If `db_path` is empty (`""`), it defaults to `filepath.Join(DefaultConfigDir(), "mods.db")`.

---

## Steam Proton Auto-Detection Algorithm

On Linux systems (including Steam Deck and SteamOS), *The Sims 4* runs inside a Proton compatibility prefix. `ts4ccmanager` implements a VDF-based auto-detection algorithm to locate the game's prefix without requiring manual path entry.

### Detection Steps

1. **Locate Steam Root**:
   - Checks `~/.steam/steam/`.
2. **Read Library Index**:
   - Opens `steamapps/libraryfolders.vdf`.
3. **Parse Valve Data Format (VDF)**:
   - Uses `github.com/andygrunwald/vdf` to parse library folders and mounted storage devices.
4. **Locate Sims 4 App ID (`1222670`)**:
   - Recursively traverses all configured Steam library mount paths until the block containing app ID `1222670` is located.
5. **Construct Proton Prefix Path**:
   - Resolves the prefix directory at `<LibraryPath>/steamapps/compatdata/1222670/`.
6. **Resolve Sims 4 Document Folder**:
   - Navigates the Wine prefix virtual Windows drive:
     `pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4/Mods`
7. **Verification**:
   - Verifies the `Mods` directory exists on the filesystem and returns the resolved absolute path.

---

## Configuration Commands

### Viewing Active Configuration
```bash
tscc config show
```
Example output:
```text
CONFIG FILE:                    /home/user/.config/ts4ccmanager/config.json
MODS DIRECTORY (CONFIGURED):    
MODS DIRECTORY (RESOLVED):      /home/user/.steam/steam/steamapps/compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4/Mods
DB PATH (CONFIGURED):           
DB PATH (RESOLVED):             /home/user/.config/ts4ccmanager/mods.db
```

### Reading a Specific Key
```bash
tscc config get mods_dir
tscc config get db_path
```

### Setting a Key
```bash
tscc config set mods_dir "/home/user/Games/The Sims 4/Mods"
tscc config set db_path "/home/user/Documents/ts4_backup.db"
```

---

## Platform Path Examples

| Platform / Setup | Example Mods Directory Path |
| :--- | :--- |
| **Linux (Steam / Proton)** | `~/.steam/steam/steamapps/compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4/Mods` |
| **Linux (Lutris / Wine)** | `~/Games/the-sims-4/drive_c/users/<user>/Documents/Electronic Arts/The Sims 4/Mods` |
| **Windows (Native)** | `C:\Users\<Username>\Documents\Electronic Arts\The Sims 4\Mods` |
| **macOS (Native)** | `/Users/<Username>/Documents/Electronic Arts/The Sims 4/Mods` |
