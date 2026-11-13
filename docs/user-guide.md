# ts4ccmanager User Guide

This guide provides step-by-step instructions for managing *The Sims 4* mods and custom content using `ts4ccmanager`. It covers everyday player workflows from initial setup to installing, updating, and removing mods.

---

## Table of Contents
1. [Overview](#overview)
2. [Prerequisites & Initial Setup](#prerequisites--initial-setup)
3. [Installing Mods](#installing-mods)
   - [Single `.package` Files](#single-package-files)
   - [Script Mods (`.ts4script`)](#script-mods-ts4script)
   - [Compressed Archives (`.zip`)](#compressed-archives-zip)
   - [Adding Metadata (Name, Version, Author)](#adding-metadata-name-version-author)
   - [Duplicate Prevention](#duplicate-prevention)
4. [Viewing Installed Content](#viewing-installed-content)
   - [Standard Table Output](#standard-table-output)
   - [JSON Output for Automation](#json-output-for-automation)
5. [Updating Mods](#updating-mods)
6. [Removing Mods](#removing-mods)
7. [Sims 4 Modding Best Practices](#sims-4-modding-best-practices)
8. [Troubleshooting & FAQ](#troubleshooting--faq)

---

## Overview

Managing Sims 4 custom content manually often leads to disorganized `Mods` folders, duplicate files taking up disk space, broken game scripts, and leftover assets after uninstalling. `ts4ccmanager` provides a streamlined command-line interface that:
- Copies or extracts mods directly to your game's `Mods` directory.
- Calculates and indexes SHA256 checksums to avoid installing the same content twice.
- Stores metadata (mod name, author, version, install date, active status) in a lightweight SQLite database.
- Completely removes all associated files when you uninstall a mod.

To open the placeholder terminal interface, run `tscc -t` (or `tscc --tui`).

---

## Prerequisites & Initial Setup

### 1. Verify Mod Directory Configuration

Before installing mods, verify that `ts4ccmanager` has located your game's `Mods` directory:

```bash
ts4ccmanager config show
```

Output example:
```text
CONFIG FILE:                    /home/user/.config/ts4ccmanager/config.json
MODS DIRECTORY (CONFIGURED):    
MODS DIRECTORY (RESOLVED):      /home/user/.steam/steam/steamapps/compatdata/1222670/pfx/drive_c/users/steamuser/Documents/Electronic Arts/The Sims 4/Mods
DB PATH (CONFIGURED):           
DB PATH (RESOLVED):             /home/user/.config/ts4ccmanager/mods.db
```

### 2. Setting a Custom Mods Directory (If Needed)

If automatic detection does not find your game folder (e.g. non-Steam installations, custom Steam library location, or native Windows / macOS paths), configure it manually:

```bash
# Linux native / Wine prefix
ts4ccmanager config set mods_dir "/path/to/Electronic Arts/The Sims 4/Mods"

# Windows (Command Prompt / PowerShell)
ts4ccmanager config set mods_dir "C:\Users\<User>\Documents\Electronic Arts\The Sims 4\Mods"

# macOS
ts4ccmanager config set mods_dir "/Users/<User>/Documents/Electronic Arts/The Sims 4/Mods"
```

---

## Installing Mods

The `add` command handles files and archives in one consistent command:
```bash
ts4ccmanager add <path-to-file> [flags]
```

### Single `.package` Files
`.package` files contain 3D meshes, textures, CAS items, furniture, overrides, or tuning mods.

```bash
ts4ccmanager add ~/Downloads/boho_dress.package
```
*Result:* The file is copied directly into your `Mods` directory and registered in the database.

### Script Mods (`.ts4script`)
`.ts4script` files contain compiled Python code for complex gameplay mods and UI extensions.

```bash
ts4ccmanager add ~/Downloads/custom_interactions.ts4script
```
*Result:* The script mod is placed directly in the `Mods` root directory to ensure The Sims 4 game engine can execute it.

### Compressed Archives (`.zip`)
Many creators distribute mod sets or multi-part mods in `.zip` archives. `ts4ccmanager` extracts only `.package` and `.ts4script` files directly into the `Mods` root; paths inside the archive are flattened to filenames.

```bash
ts4ccmanager add ~/Downloads/kitchen_set.zip
```
*Result:* Copies supported files such as `sofa.package` and `scripts/gameplay.ts4script` to `Mods/sofa.package` and `Mods/gameplay.ts4script`, then tracks all files with the archive's mod record. If two archive entries have the same filename, or a destination already exists, installation fails rather than overwriting it.

### Adding Metadata (Name, Version, Author)
You can attach human-readable metadata when installing a mod:

```bash
ts4ccmanager add ~/Downloads/cas_eyelashes.package \
  --name "3D Eyelashes V4" \
  --author "Kijiko" \
  --version "4.1"
```

Supported flags:
- `-n, --name string`: Custom display name (defaults to the file basename).
- `-v, --version string`: Version number or release tag.
- `-a, --author string`: Mod author or creator name.

### Duplicate Prevention
`ts4ccmanager` computes a SHA256 checksum for each file before copying or extracting. If an identical mod file is already installed, the command prevents duplicate installation:

```bash
ts4ccmanager add ~/Downloads/duplicate_item.package
# Error: mod is already installed (duplicate checksum found): Duplicate Item (ID: 4)
```

---

## Viewing Installed Content

### Standard Table Output
Run `ts4ccmanager list` to view all tracked mods:

```bash
ts4ccmanager list
```

Output:
```text
ID   NAME                FILE NAME             VERSION   AUTHOR    STATUS   INSTALLED AT
1    3D Eyelashes V4     cas_eyelashes.package 4.1       Kijiko    active   2026-10-29 10:15:20
2    Gameplay Overhaul   gameplay_overhaul     2.0       Lumia     active   2026-10-29 11:42:00
3    Living Room Set     living_room_set       1.0       Peacemaker active  2026-10-29 12:05:14
```

### JSON Output for Automation
For scripting, backup verification, or integrating with other tools, use the `--json` flag:

```bash
ts4ccmanager list --json
```

Output:
```json
[
  {
    "id": 1,
    "name": "3D Eyelashes V4",
    "file_name": "cas_eyelashes.package",
    "file_path": "/path/to/Mods/cas_eyelashes.package",
    "file_hash": "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e",
    "version": "4.1",
    "author": "Kijiko",
    "installed_at": "2026-10-29T10:15:20Z",
    "status": "active"
  }
]
```

---

## Updating Mods

When a mod creator releases a new version, update it using either its numeric ID or its name:

```bash
# Update by name
ts4ccmanager update "Gameplay Overhaul" ~/Downloads/gameplay_overhaul_v2_1.zip --version "2.1"

# Update by ID
ts4ccmanager update 1 ~/Downloads/cas_eyelashes_v5.package --version "5.0"
```

What happens during an update:
1. The replacement file or archive is staged, and the mod's currently tracked files are backed up.
2. The new file or supported archive entries are installed in the `Mods` root.
3. The database record is updated; if the update fails, the previous files are restored.

---

## Removing Mods

To uninstall a mod and delete all associated files from disk:

```bash
# Remove by name
ts4ccmanager remove "Living Room Set"

# Remove by ID
ts4ccmanager remove 3
```

`ts4ccmanager` removes the file or folder from the game directory and cleans up the SQLite record, guaranteeing no orphaned files remain.

---

## Sims 4 Modding Best Practices

1. **Script Mod Depth Limit**: *The Sims 4* engine only loads `.ts4script` files if they are located directly in the root of the `Mods/` folder or at most **one subfolder deep** (`Mods/ModName/`). `ts4ccmanager` installs supported archive files directly in the `Mods` root.
2. **Game Patch Preparation**: Before a major game update, export your installed mod list using `ts4ccmanager list --json > installed_mods.json` so you have a complete inventory of what needs updating.
3. **In-Game Settings**: Remember to enable both **"Enable Custom Content and Mods"** and **"Script Mods Allowed"** in The Sims 4 Game Options -> Other menu after game updates.

---

## Troubleshooting & FAQ

### "mods directory is not configured and automatic Steam detection failed"
- **Cause**: Steam Proton installation was not found at standard paths.
- **Solution**: Manually configure your mods folder path:
  ```bash
  ts4ccmanager config set mods_dir "/path/to/The Sims 4/Mods"
  ```

### "mod is already installed (duplicate checksum found)"
- **Cause**: You tried to install a file that matches an existing mod's SHA256 hash.
- **Solution**: Check `ts4ccmanager list` to find the existing entry. If you intended to replace it with a new release, use `ts4ccmanager update <id|name> <new-file>`.

### "unsupported file format"
- **Cause**: The file extension is not `.package`, `.ts4script`, or `.zip`.
- **Solution**: Ensure your downloaded mod is an uncorrupted `.package`, `.ts4script`, or standard `.zip` file.
