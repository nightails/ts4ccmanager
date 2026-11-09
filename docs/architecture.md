# ts4ccmanager System Architecture

This document outlines the technical design, architectural patterns, database schema, and security considerations of `ts4ccmanager`.

---

## Architectural Principles

1. **Zero CGO & Portability**: Built with `modernc.org/sqlite` (a pure Go SQLite implementation) to allow single-binary cross-compilation across Linux, macOS, and Windows without external C compiler toolchain dependencies.
2. **Layered Separation of Concerns**: Clean boundaries between the CLI presentation layer (`cmd`), domain business logic (`internal/mod`), persistence (`internal/db`), platform discovery (`internal/detector`), and configuration (`internal/config`).
3. **Transactional Integrity & Rollback**: Filesystem mutations and database operations are orchestrated such that failures roll back filesystem changes, preventing orphaned files or database inconsistencies.
4. **Security by Default**: Proactive defense against malicious archives via Zip Slip path traversal checks and checksum indexing.

---

## High-Level Architecture Diagram

```text
+-------------------------------------------------------------+
|                         CLI Layer                           |
|      (cmd/root.go, cmd/add.go, cmd/list.go, cmd/config.go)  |
+------------------------------+------------------------------+
                               |
                               v
+-------------------------------------------------------------+
|                     Domain Service Layer                    |
|                   (internal/mod/service.go)                 |
+--------------+------------------------------+---------------+
               |                              |
               v                              v
+-----------------------------+  +----------------------------+
|      Persistence Layer      |  |      File Operations       |
|  (internal/db/repository.go)|  | (internal/mod/extractor.go |
|   backed by SQLite Driver   |  |  internal/mod/file_mgr.go) |
+-----------------------------+  +----------------------------+
               ^                              ^
               |                              |
+--------------+------------------------------+---------------+
|                    Configuration & Platform                 |
|       (internal/config/config.go, internal/detector/steam.go)|
+-------------------------------------------------------------+
```

---

## Package Breakdown & Responsibilities

### 1. `cmd/`
- **Purpose**: CLI command declarations, argument parsing, flag bindings, and terminal formatting.
- **Framework**: `github.com/spf13/cobra`.
- **Output Rendering**: Uses Go's `text/tabwriter` for aligned tabular display and `encoding/json` for machine-readable output.

### 2. `internal/config/`
- **Purpose**: Configuration data structures and persistence logic.
- **Key Types**:
  - `Config`: Struct with `ModsDir` and `DBPath`.
- **Key Methods**:
  - `Load()` / `LoadFrom(path)`: Reads JSON configuration from disk.
  - `Save()` / `SaveTo(path)`: Writes JSON configuration to disk.
  - `ResolveModsDir()`: Returns explicit path or triggers platform detection.
  - `ResolveDBPath()`: Returns explicit path or default `~/.config/ts4ccmanager/mods.db`.

### 3. `internal/detector/`
- **Purpose**: Automatic discovery of game folders in Linux Steam Proton installations.
- **Key Component**: Valve Data Format (`.vdf`) parser (`github.com/andygrunwald/vdf`).
- **Detection Target**: Sims 4 App ID `1222670`.

### 4. `internal/db/`
- **Purpose**: Database connection management, schema initialization, and CRUD operations.
- **Key Types**:
  - `Mod`: Entity representing a tracked mod.
  - `Repository`: Interface defining data persistence contracts.
  - `SQLiteRepository`: Concrete implementation wrapping `*sql.DB`.

### 5. `internal/mod/`
- **Purpose**: Core domain orchestration.
- **Components**:
  - `Service`: Coordinates between repository and filesystem operations.
  - `FileManager`: File copying, existence checks, SHA256 hashing (`CalculateFileHash`), and deletion.
  - `Extractor`: Format validation and secure archive extraction (`ExtractZip`).

---

## Relational Database Schema & Data Models

### Database Schema
On startup, `ts4ccmanager` creates the schema below and adds `installed_files` to existing databases when needed:

```sql
CREATE TABLE IF NOT EXISTS mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    installed_files TEXT NOT NULL DEFAULT '[]',
    file_hash TEXT NOT NULL,
    version TEXT DEFAULT '',
    author TEXT DEFAULT '',
    installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_mods_name ON mods(name);
CREATE INDEX IF NOT EXISTS idx_mods_file_hash ON mods(file_hash);
```

### Mod Entity Mapping

```go
type Mod struct {
    ID          int64     `json:"id"`
    Name        string    `json:"name"`
    FileName    string    `json:"file_name"`
    FilePath    string    `json:"file_path"`
    InstalledFiles []string `json:"installed_files,omitempty"`
    FileHash    string    `json:"file_hash"`
    Version     string    `json:"version"`
    Author      string    `json:"author"`
    InstalledAt time.Time `json:"installed_at"`
    Status      string    `json:"status"` // "active" or "disabled"
}
```

### Schema Constraints & Invariants
- `file_path UNIQUE`: Ensures two mods cannot occupy the exact same target path on disk.
- `installed_files`: Stores the paths owned by archive installs; old records without this list continue to use `file_path`.
- The service checks every flattened destination before installation and rejects collisions rather than overwriting files it does not own.
- `idx_mods_file_hash`: Enables O(1) duplicate detection when checking if a file's SHA256 checksum has already been installed.
- `idx_mods_name`: Optimizes lookup operations when resolving mods by name instead of numeric ID.

---

## Operational Workflows & Atomicity

### 1. Installation (`Add`)

```text
[Input File]
     |
     v
1. Validate File Format (.package, .ts4script, .zip)
     |
     v
2. Calculate SHA256 Checksum
     |
     v
3. Query DB by Hash -> If exists, reject with ErrDuplicateMod
     |
     v
4. Stage files (copy a mod or flatten supported Zip entries), then install into the Mods root without overwriting existing files
     |
     +---> [Failure] -> Abort (No disk remnants)
     |
     v [Success]
5. Insert Record into SQLite DB
     |
     +---> [DB Failure] -> Roll back by deleting created file(s)
     |
     v [Success]
[Return Registered Mod]
```

### 2. Updating (`Update`)
1. Resolves target mod record by ID or Name.
2. Validates new file format and calculates new SHA256 checksum.
3. Stages the replacement and backs up paths owned by the existing record.
4. Installs the new file or flattened archive entries in the Mods root.
5. Updates SQLite; on failure, removes new files and restores the backups.

### 3. Deletion (`Remove`)
1. Resolves target mod record by ID or Name.
2. Deletes the tracked file(s) from the `Mods` folder (or the legacy directory referenced by an older record).
3. Deletes the database row from `mods`.

---

## Security Considerations

### Zip Slip Vulnerability Prevention
When extracting multi-file `.zip` archives, malicious archives could contain relative path traversal sequences (such as `../../etc/passwd` or `../harmful_file`).

`ts4ccmanager` sanitizes all zip entries before extraction:
```go
targetPath := filepath.Join(cleanDestDir, f.Name)
cleanTarget := filepath.Clean(targetPath)

// Security check: zip-slip prevention
if !strings.HasPrefix(cleanTarget, cleanDestDir+string(os.PathSeparator)) && cleanTarget != cleanDestDir {
    return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
}
```
If an entry attempts to escape the designated destination directory, extraction immediately halts with an error and removes partial files.

### Checksum Verification
SHA256 hashes are calculated via streaming `io.Copy` into `crypto/sha256.New()`, avoiding high memory usage for large package files while guaranteeing file integrity and duplicate prevention.
