package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) Repository {
	database, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	return NewRepository(database)
}

func TestRepositoryCRUD(t *testing.T) {
	repo := setupTestDB(t)

	mod := &Mod{
		Name:     "Awesome Hair",
		FileName: "awesome_hair.package",
		FilePath: "/sims4/mods/awesome_hair.package",
		FileHash: "hash123456789",
		InstalledFiles: []string{
			"/sims4/mods/awesome_hair.package",
			"/sims4/mods/awesome_hair.ts4script",
		},
		Version: "1.0.0",
		Author:  "CreatorA",
		Status:  "active",
	}

	// 1. Test Insert
	if err := repo.InsertMod(mod); err != nil {
		t.Fatalf("failed to insert mod: %v", err)
	}
	if mod.ID == 0 {
		t.Errorf("expected non-zero mod ID after insert")
	}
	if mod.InstalledAt.IsZero() {
		t.Errorf("expected non-zero InstalledAt timestamp")
	}

	// 2. Test GetByID
	fetched, err := repo.GetModByID(mod.ID)
	if err != nil {
		t.Fatalf("failed to get mod by ID: %v", err)
	}
	if fetched.Name != mod.Name || fetched.FileHash != mod.FileHash || fetched.Author != mod.Author || !reflect.DeepEqual(fetched.InstalledFiles, mod.InstalledFiles) {
		t.Errorf("fetched mod does not match inserted mod: %+v vs %+v", fetched, mod)
	}

	// 3. Test GetByName
	byName, err := repo.GetModByName("Awesome Hair")
	if err != nil {
		t.Fatalf("failed to get mod by Name: %v", err)
	}
	if byName.ID != mod.ID {
		t.Errorf("expected mod ID %d, got %d", mod.ID, byName.ID)
	}

	// 4. Test GetByPath
	byPath, err := repo.GetModByPath("/sims4/mods/awesome_hair.package")
	if err != nil {
		t.Fatalf("failed to get mod by Path: %v", err)
	}
	if byPath.ID != mod.ID {
		t.Errorf("expected mod ID %d, got %d", mod.ID, byPath.ID)
	}

	// 5. Test GetByHash
	byHash, err := repo.GetModByHash("hash123456789")
	if err != nil {
		t.Fatalf("failed to get mod by Hash: %v", err)
	}
	if byHash.ID != mod.ID {
		t.Errorf("expected mod ID %d, got %d", mod.ID, byHash.ID)
	}

	// 6. Test FindModByIDOrName
	byIDStr, err := repo.FindModByIDOrName("1")
	if err != nil || byIDStr.ID != mod.ID {
		t.Errorf("failed to find mod by numeric ID string: %v", err)
	}

	byNameStr, err := repo.FindModByIDOrName("Awesome Hair")
	if err != nil || byNameStr.ID != mod.ID {
		t.Errorf("failed to find mod by name string: %v", err)
	}

	// 7. Test ListMods
	mod2 := &Mod{
		Name:        "Cool Clothes",
		FileName:    "cool_clothes.package",
		FilePath:    "/sims4/mods/cool_clothes.package",
		FileHash:    "hash987654321",
		Version:     "2.0.0",
		Author:      "CreatorB",
		InstalledAt: time.Now().UTC(),
		Status:      "active",
	}
	if err := repo.InsertMod(mod2); err != nil {
		t.Fatalf("failed to insert mod2: %v", err)
	}

	allMods, err := repo.ListMods()
	if err != nil {
		t.Fatalf("failed to list mods: %v", err)
	}
	if len(allMods) != 2 {
		t.Fatalf("expected 2 mods, got %d", len(allMods))
	}

	// 8. Test UpdateMod
	mod.Version = "1.1.0"
	mod.Name = "Awesome Hair V2"
	if err := repo.UpdateMod(mod); err != nil {
		t.Fatalf("failed to update mod: %v", err)
	}

	updated, err := repo.GetModByID(mod.ID)
	if err != nil {
		t.Fatalf("failed to get updated mod: %v", err)
	}
	if updated.Version != "1.1.0" || updated.Name != "Awesome Hair V2" {
		t.Errorf("mod update was not persisted: %+v", updated)
	}

	// 9. Test DeleteMod
	if err := repo.DeleteMod(mod.ID); err != nil {
		t.Fatalf("failed to delete mod: %v", err)
	}

	_, err = repo.GetModByID(mod.ID)
	if !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound after deletion, got %v", err)
	}

	remainingMods, err := repo.ListMods()
	if err != nil {
		t.Fatalf("failed to list mods after deletion: %v", err)
	}
	if len(remainingMods) != 1 {
		t.Errorf("expected 1 remaining mod, got %d", len(remainingMods))
	}
}

func TestOpenMigratesLegacyModsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create legacy database: %v", err)
	}
	_, err = legacyDB.Exec(`
		CREATE TABLE mods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_path TEXT NOT NULL UNIQUE,
			file_hash TEXT NOT NULL,
			version TEXT DEFAULT '',
			author TEXT DEFAULT '',
			installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'active'
		);
		INSERT INTO mods (name, file_name, file_path, file_hash) VALUES ('Legacy', 'legacy_pack', '/mods/legacy_pack', 'legacy-hash');
	`)
	if err != nil {
		_ = legacyDB.Close()
		t.Fatalf("failed to prepare legacy schema: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("failed to close legacy database: %v", err)
	}

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to migrate legacy database: %v", err)
	}
	defer database.Close()

	mod, err := NewRepository(database).GetModByID(1)
	if err != nil {
		t.Fatalf("failed to read legacy mod after migration: %v", err)
	}
	if mod.FilePath != "/mods/legacy_pack" || len(mod.InstalledFiles) != 0 {
		t.Errorf("legacy record was not preserved with empty installed-file list: %+v", mod)
	}

	var appliedMigrations int
	if err := database.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1 AND is_applied = 1").Scan(&appliedMigrations); err != nil {
		t.Fatalf("failed to verify Goose migration: %v", err)
	}
	if appliedMigrations != 1 {
		t.Errorf("expected initial Goose migration to be recorded, got %d applied migrations", appliedMigrations)
	}
}

func TestRepositoryNotFoundErrors(t *testing.T) {
	repo := setupTestDB(t)

	if _, err := repo.GetModByID(999); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound, got %v", err)
	}

	if _, err := repo.GetModByName("NonExistent"); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound, got %v", err)
	}

	if _, err := repo.GetModByPath("/non/existent"); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound, got %v", err)
	}

	if _, err := repo.GetModByHash("nonexistenthash"); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound, got %v", err)
	}

	if _, err := repo.FindModByIDOrName("999"); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound, got %v", err)
	}

	if err := repo.DeleteMod(999); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound on delete, got %v", err)
	}

	mod := &Mod{ID: 999, Name: "Test", FilePath: "/test", FileHash: "hash"}
	if err := repo.UpdateMod(mod); !errors.Is(err, ErrModNotFound) {
		t.Errorf("expected ErrModNotFound on update, got %v", err)
	}
}
