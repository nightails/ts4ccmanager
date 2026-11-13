# ts4ccmanager Development & Contributing Guide

This guide provides instructions for developers contributing to `ts4ccmanager`, including environment setup, coding patterns, testing strategies, and guidelines for adding new commands.

---

## Development Prerequisites

- **Go**: Version 1.22 or higher (pure Go, no CGO toolchain required).
- **Git**: For version control.

---

## Project Structure

```text
ts4ccmanager/
├── cmd/                          # Presentation layer (Cobra commands)
│   ├── root.go                   # Root command and dependency initializers
│   ├── add.go                    # 'add' command handler
│   ├── list.go                   # 'list' command handler
│   ├── remove.go                 # 'remove' command handler
│   ├── update.go                 # 'update' command handler
│   ├── config.go                 # 'config' subcommand group
│   └── cmd_test.go               # End-to-end CLI workflow tests
├── internal/                     # Private application packages
│   ├── config/                   # Configuration persistence and resolution
│   │   ├── config.go
│   │   └── config_test.go
│   ├── db/                       # Database models, schema, and repository
│   │   ├── db.go                 # Connection lifecycle and migrations
│   │   ├── models.go             # Domain data entity
│   │   ├── repository.go         # Repository mapping to generated queries
│   │   ├── db.sql.go             # sqlc query runner
│   │   ├── models.sql.go         # sqlc database records
│   │   ├── query.sql.go          # sqlc-generated query methods
│   │   ├── sql/                  # sqlc schema and named queries
│   │   ├── migrations/           # Embedded Goose migrations
│   │   └── repository_test.go    # In-memory database test suite
│   ├── detector/                 # Steam Proton path detection
│   │   ├── steam.go              # VDF parsing and path resolution
│   │   └── steam_test.go         # VDF tree traversal tests
│   └── mod/                      # Domain logic
│       ├── extractor.go          # Zip extraction & format checks
│       ├── file_manager.go       # Checksum calculation & disk I/O
│       ├── service.go            # High-level orchestrator
│       └── service_test.go       # Domain service tests
├── docs/                         # Project documentation
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── main.go                       # Application entrypoint
└── README.md                     # Project landing page
```

---

## Building from Source

To compile the application binary:

```bash
# Build the binary in the project root
go build -o tscc ./cmd/tscc

# Run directly without building a separate binary
go run ./cmd/tscc --help
```

---

## Testing Strategy & Best Practices

All packages in `ts4ccmanager` are covered by automated unit and integration tests.

### Running Tests

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests with race condition detector
go test -race ./...

# Check test coverage
go test -cover ./...
```

### Key Testing Patterns Used

1. **In-Memory SQLite Isolation**:
   Database tests use SQLite in-memory mode (`:memory:`), providing fast, hermetic test execution without disk cleanup overhead:
   ```go
   database, err := db.Open(":memory:")
   if err != nil {
       t.Fatalf("failed to open in-memory db: %v", err)
   }
   t.Cleanup(func() { _ = database.Close() })
   ```

2. **Temporary Directories for Filesystem Isolation**:
   File operations and CLI tests use `t.TempDir()` or `os.MkdirTemp` with `t.Cleanup` or `defer os.RemoveAll(tempDir)` to isolate disk writes.

3. **CLI Buffer Redirection**:
   CLI tests redirect `rootCmd.SetOut(buf)` and `rootCmd.SetErr(buf)` to inspect output strings and verify stdout formatting without printing to terminal streams.

### Database Schema Workflow

- Edit `internal/db/sql/schema.sql` and keep it consistent with the numbered migrations in `internal/db/migrations/`.
- Define database operations as named queries in `internal/db/sql/query.sql`.
- Run `sqlc generate` from the project root to update the generated Go files in `internal/db/`; do not edit generated files by hand.
- `db.Open` embeds and applies Goose migrations automatically. Preserve the legacy `installed_files` upgrade in `internal/db/db.go` for databases created before Goose was introduced.

---

## Code Style & Conventions

- **Zero CGO**: Always rely on `modernc.org/sqlite` instead of `github.com/mattn/go-sqlite3` to maintain pure Go cross-compilation.
- **Error Wrapping**: Wrap lower-level errors with context using `fmt.Errorf("action description: %w", err)`.
- **Atomic Operations**: When modifying both filesystem and database, always clean up disk changes if the database operation fails.
- **Security Validations**: Any file path received from archives or user input must be sanitized with `filepath.Clean` and validated against directory escapes.

---

## Adding a New Command

To add a new CLI command to `ts4ccmanager`:

1. **Implement Domain Logic in `internal/`**:
   Add any new repository methods to `internal/db/repository.go` and orchestration functions to `internal/mod/service.go`. Add unit tests in the respective `_test.go` files.
2. **Define Cobra Command in `cmd/`**:
   Create a new file (e.g. `cmd/toggle.go`):
   ```go
   package cmd

   import (
       "fmt"
       "github.com/spf13/cobra"
   )

   var toggleCmd = &cobra.Command{
       Use:   "toggle <id|name>",
       Short: "Enable or disable a mod",
       Args:  cobra.ExactArgs(1),
       RunE: func(cmd *cobra.Command, args []string) error {
           svc, repo, _, err := initModService()
           if err != nil {
               return err
           }
           defer repo.Close()

           // Call service logic
           return nil
       },
   }

   func init() {
       rootCmd.AddCommand(toggleCmd)
   }
   ```
3. **Add Integration Tests in `../internal/cli/cmd_test.go`**:
   Verify CLI flag parsing, execution, and output handling.
4. **Update Documentation**:
   Update `docs/cli-reference.md` and `docs/user-guide.md` to document the new command.
