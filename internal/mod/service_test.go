package mod

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ts4ccmanager/internal/db"
)

func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
		t.Fatalf("failed to create zip parent dir: %v", err)
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to add file to zip: %v", err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write content to zip: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
}

func setupTestService(t *testing.T) (*Service, db.Repository, string, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ts4ccmanager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	modsDir := filepath.Join(tempDir, "Mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		t.Fatalf("failed to create mods dir: %v", err)
	}

	srcDir := filepath.Join(tempDir, "downloads")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	repo := db.NewRepository(database)
	svc := NewService(repo, modsDir)

	t.Cleanup(func() {
		_ = database.Close()
		_ = os.RemoveAll(tempDir)
	})

	return svc, repo, modsDir, srcDir
}

type failingRepository struct {
	db.Repository
	insertErr error
	updateErr error
	deleteErr error
}

func (r failingRepository) InsertMod(mod *db.Mod) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	return r.Repository.InsertMod(mod)
}

func (r failingRepository) UpdateMod(mod *db.Mod) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.Repository.UpdateMod(mod)
}

func (r failingRepository) DeleteMod(id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return r.Repository.DeleteMod(id)
}

func TestServiceAddPackage(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)

	pkgPath := filepath.Join(srcDir, "test_hair.package")
	if err := os.WriteFile(pkgPath, []byte("package-content-hair-1"), 0644); err != nil {
		t.Fatalf("failed to create dummy package: %v", err)
	}

	mod, err := svc.Add(pkgPath, AddOptions{
		Name:    "Custom Hair Mod",
		Version: "1.0",
		Author:  "Modder1",
	})
	if err != nil {
		t.Fatalf("unexpected error adding package: %v", err)
	}

	if mod.ID == 0 {
		t.Errorf("expected valid mod ID, got 0")
	}
	if mod.Name != "Custom Hair Mod" {
		t.Errorf("expected mod name 'Custom Hair Mod', got %q", mod.Name)
	}
	if mod.FileName != "test_hair.package" {
		t.Errorf("expected file name 'test_hair.package', got %q", mod.FileName)
	}

	destPath := filepath.Join(modsDir, "test_hair.package")
	if !PathExists(destPath) {
		t.Errorf("expected file to exist at %s", destPath)
	}

	// Test Duplicate Error
	_, err = svc.Add(pkgPath, AddOptions{})
	if err == nil || !errors.Is(err, ErrDuplicateMod) {
		t.Errorf("expected ErrDuplicateMod when adding identical file, got: %v", err)
	}
}

func TestServiceAddTS4Script(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)

	scriptPath := filepath.Join(srcDir, "cool_script.ts4script")
	if err := os.WriteFile(scriptPath, []byte("script-content-binary"), 0644); err != nil {
		t.Fatalf("failed to create dummy script: %v", err)
	}

	mod, err := svc.Add(scriptPath, AddOptions{})
	if err != nil {
		t.Fatalf("unexpected error adding script: %v", err)
	}

	if mod.Name != "cool_script" {
		t.Errorf("expected mod name 'cool_script', got %q", mod.Name)
	}

	destPath := filepath.Join(modsDir, "cool_script.ts4script")
	if !PathExists(destPath) {
		t.Errorf("expected file to exist at %s", destPath)
	}
}

func TestServiceAddZipArchive(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)

	zipPath := filepath.Join(srcDir, "furniture_pack.zip")
	createTestZip(t, zipPath, map[string]string{
		"sofa.package":  "sofa-content",
		"table.package": "table-content",
	})

	mod, err := svc.Add(zipPath, AddOptions{
		Name: "Furniture Pack",
	})
	if err != nil {
		t.Fatalf("unexpected error adding zip: %v", err)
	}

	if mod.Name != "Furniture Pack" {
		t.Errorf("expected mod name 'Furniture Pack', got %q", mod.Name)
	}

	if !PathExists(filepath.Join(modsDir, "sofa.package")) {
		t.Errorf("expected extracted sofa.package in Mods root")
	}
	if !PathExists(filepath.Join(modsDir, "table.package")) {
		t.Errorf("expected extracted table.package in Mods root")
	}
	if mod.FilePath != filepath.Join(modsDir, "sofa.package") && mod.FilePath != filepath.Join(modsDir, "table.package") {
		t.Errorf("expected archive record path to reference an installed file, got %q", mod.FilePath)
	}
	if len(mod.InstalledFiles) != 2 {
		t.Errorf("expected both extracted files to be tracked, got %v", mod.InstalledFiles)
	}
}

func TestServiceAddZipWithoutSupportedFiles(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)
	zipPath := filepath.Join(srcDir, "unsupported_pack.zip")
	createTestZip(t, zipPath, map[string]string{
		"readme.txt":         "readme-content",
		"nested/archive.zip": "nested-zip-content",
	})

	_, err := svc.Add(zipPath, AddOptions{})
	if err == nil || err.Error() != "zip archive contains no supported files: "+zipPath {
		t.Fatalf("expected no supported files error, got: %v", err)
	}
	entries, readErr := os.ReadDir(modsDir)
	if readErr != nil {
		t.Fatalf("failed to read mods directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected failed extraction to leave Mods untouched, found %v", entries)
	}
}

func TestServiceAddZipFlattensNestedEntriesAndRemoveCleansAllFiles(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)
	zipPath := filepath.Join(srcDir, "nested_pack.zip")
	createTestZip(t, zipPath, map[string]string{
		"outer/first.package":       "first-content",
		"outer/deep/second.package": "second-content",
	})

	mod, err := svc.Add(zipPath, AddOptions{})
	if err != nil {
		t.Fatalf("failed to add archive: %v", err)
	}
	for name, content := range map[string]string{"first.package": "first-content", "second.package": "second-content"} {
		got, err := os.ReadFile(filepath.Join(modsDir, name))
		if err != nil || string(got) != content {
			t.Errorf("unexpected installed content for %s: %q, %v", name, got, err)
		}
	}
	if err := svc.Remove(mod.Name); err != nil {
		t.Fatalf("failed to remove archive mod: %v", err)
	}
	for _, name := range []string{"first.package", "second.package"} {
		if PathExists(filepath.Join(modsDir, name)) {
			t.Errorf("expected %s to be removed", name)
		}
	}
}

func TestServiceAddZipRejectsFilenameCollisionsWithoutOverwriting(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)
	existing := filepath.Join(modsDir, "shared.package")
	if err := os.WriteFile(existing, []byte("existing-content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}
	zipPath := filepath.Join(srcDir, "collision.zip")
	createTestZip(t, zipPath, map[string]string{"nested/shared.package": "archive-content"})
	if _, err := svc.Add(zipPath, AddOptions{}); err == nil {
		t.Fatal("expected collision with an existing Mods file to fail")
	}
	got, err := os.ReadFile(existing)
	if err != nil || string(got) != "existing-content" {
		t.Fatalf("existing Mods file was changed: %q, %v", got, err)
	}

	duplicateZip := filepath.Join(srcDir, "duplicate_names.zip")
	createTestZip(t, duplicateZip, map[string]string{
		"one/shared.package": "first",
		"two/shared.package": "second",
	})
	if _, err := svc.Add(duplicateZip, AddOptions{}); err == nil {
		t.Fatal("expected duplicate archive basenames to fail")
	}
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		t.Fatalf("failed to read mods directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "shared.package" {
		t.Errorf("failed archive installs changed Mods contents: %v", entries)
	}
}

func TestServiceAddZipRollsBackWhenDatabaseInsertFails(t *testing.T) {
	svc, repo, modsDir, srcDir := setupTestService(t)
	svc.repo = failingRepository{Repository: repo, insertErr: errors.New("insert failed")}
	zipPath := filepath.Join(srcDir, "rollback.zip")
	createTestZip(t, zipPath, map[string]string{"one.package": "one", "two.package": "two"})

	if _, err := svc.Add(zipPath, AddOptions{}); err == nil {
		t.Fatal("expected database insert failure")
	}
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		t.Fatalf("failed to read Mods directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("failed add left files in Mods: %v", entries)
	}
}

func TestServiceUnsupportedFormat(t *testing.T) {
	svc, _, _, srcDir := setupTestService(t)

	txtPath := filepath.Join(srcDir, "readme.txt")
	_ = os.WriteFile(txtPath, []byte("not a mod"), 0644)

	_, err := svc.Add(txtPath, AddOptions{})
	if err == nil || !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("expected ErrUnsupportedFormat, got: %v", err)
	}
}

func TestServiceListAndGet(t *testing.T) {
	svc, _, _, srcDir := setupTestService(t)

	pkg1 := filepath.Join(srcDir, "mod1.package")
	_ = os.WriteFile(pkg1, []byte("mod1"), 0644)
	pkg2 := filepath.Join(srcDir, "mod2.package")
	_ = os.WriteFile(pkg2, []byte("mod2"), 0644)

	m1, _ := svc.Add(pkg1, AddOptions{Name: "Mod One"})
	m2, _ := svc.Add(pkg2, AddOptions{Name: "Mod Two"})

	list, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error listing mods: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 mods, got %d", len(list))
	}

	fetched1, err := svc.Get("Mod One")
	if err != nil || fetched1.ID != m1.ID {
		t.Errorf("failed to get mod by name: %v", err)
	}

	fetched2, err := svc.Get(string(rune('0' + int(m2.ID))))
	if err != nil || fetched2.ID != m2.ID {
		t.Errorf("failed to get mod by ID: %v", err)
	}
}

func TestServiceUpdate(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)

	pkgV1 := filepath.Join(srcDir, "clothing.package")
	_ = os.WriteFile(pkgV1, []byte("clothing-v1"), 0644)

	mod, err := svc.Add(pkgV1, AddOptions{Name: "Clothing", Version: "1.0", Author: "DesignerA"})
	if err != nil {
		t.Fatalf("failed to add initial mod: %v", err)
	}
	if mod.ID == 0 {
		t.Fatalf("expected non-zero mod ID")
	}

	pkgV2 := filepath.Join(srcDir, "clothing_v2.package")
	_ = os.WriteFile(pkgV2, []byte("clothing-v2-new-content"), 0644)

	updatedMod, err := svc.Update("Clothing", pkgV2, UpdateOptions{
		Version: "2.0",
		Author:  "DesignerB",
	})
	if err != nil {
		t.Fatalf("failed to update mod: %v", err)
	}

	if updatedMod.Version != "2.0" || updatedMod.Author != "DesignerB" {
		t.Errorf("metadata not updated correctly: %+v", updatedMod)
	}

	if !PathExists(filepath.Join(modsDir, "clothing_v2.package")) {
		t.Errorf("expected new file to exist in mods dir")
	}
	if PathExists(filepath.Join(modsDir, "clothing.package")) {
		t.Errorf("expected old file to be removed from mods dir")
	}
}

func TestServiceUpdateZipReplacesTrackedFiles(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)
	firstZip := filepath.Join(srcDir, "pack_v1.zip")
	createTestZip(t, firstZip, map[string]string{
		"nested/shared.package": "old-shared",
		"old.package":           "old-only",
	})
	installed, err := svc.Add(firstZip, AddOptions{Name: "Pack"})
	if err != nil {
		t.Fatalf("failed to add initial archive: %v", err)
	}

	secondZip := filepath.Join(srcDir, "pack_v2.zip")
	createTestZip(t, secondZip, map[string]string{
		"other/shared.package": "new-shared",
		"new.package":          "new-only",
	})
	updated, err := svc.Update("Pack", secondZip, UpdateOptions{Version: "2"})
	if err != nil {
		t.Fatalf("failed to update archive: %v", err)
	}
	if updated.ID != installed.ID || updated.Version != "2" {
		t.Errorf("updated mod metadata not preserved: %+v", updated)
	}
	for name, content := range map[string]string{"shared.package": "new-shared", "new.package": "new-only"} {
		got, err := os.ReadFile(filepath.Join(modsDir, name))
		if err != nil || string(got) != content {
			t.Errorf("unexpected updated content for %s: %q, %v", name, got, err)
		}
	}
	if PathExists(filepath.Join(modsDir, "old.package")) {
		t.Error("obsolete archive file should be removed after update")
	}

	unsupportedZip := filepath.Join(srcDir, "unsupported.zip")
	createTestZip(t, unsupportedZip, map[string]string{"readme.txt": "text"})
	if _, err := svc.Update("Pack", unsupportedZip, UpdateOptions{}); err == nil {
		t.Fatal("expected an archive without supported files to fail update")
	}
	got, err := os.ReadFile(filepath.Join(modsDir, "shared.package"))
	if err != nil || string(got) != "new-shared" {
		t.Errorf("failed update changed existing installed file: %q, %v", got, err)
	}
}

func TestServiceUpdateRollsBackWhenDatabaseUpdateFails(t *testing.T) {
	svc, repo, modsDir, srcDir := setupTestService(t)
	firstZip := filepath.Join(srcDir, "stable.zip")
	createTestZip(t, firstZip, map[string]string{"shared.package": "original", "old.package": "old"})
	installed, err := svc.Add(firstZip, AddOptions{Name: "Stable"})
	if err != nil {
		t.Fatalf("failed to add initial archive: %v", err)
	}

	updateZip := filepath.Join(srcDir, "replacement.zip")
	createTestZip(t, updateZip, map[string]string{"shared.package": "replacement", "new.package": "new"})
	svc.repo = failingRepository{Repository: repo, updateErr: errors.New("update failed")}
	if _, err := svc.Update("Stable", updateZip, UpdateOptions{Version: "2"}); err == nil {
		t.Fatal("expected database update failure")
	}
	for name, content := range map[string]string{"shared.package": "original", "old.package": "old"} {
		got, err := os.ReadFile(filepath.Join(modsDir, name))
		if err != nil || string(got) != content {
			t.Errorf("failed update did not restore %s: %q, %v", name, got, err)
		}
	}
	if PathExists(filepath.Join(modsDir, "new.package")) {
		t.Error("new file from failed update should be removed")
	}
	stored, err := repo.GetModByID(installed.ID)
	if err != nil {
		t.Fatalf("failed to retrieve original database record: %v", err)
	}
	if stored.Version != "" || len(stored.InstalledFiles) != 2 {
		t.Errorf("failed update changed database record: %+v", stored)
	}
}

func TestServiceRemoveLegacyArchiveDirectory(t *testing.T) {
	svc, repo, modsDir, _ := setupTestService(t)
	legacyDir := filepath.Join(modsDir, "legacy_pack")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy archive directory: %v", err)
	}
	legacyFile := filepath.Join(legacyDir, "old.package")
	if err := os.WriteFile(legacyFile, []byte("legacy"), 0644); err != nil {
		t.Fatalf("failed to create legacy archive file: %v", err)
	}
	legacyMod := &db.Mod{Name: "Legacy", FileName: "legacy_pack", FilePath: legacyDir, FileHash: "legacy-hash"}
	if err := repo.InsertMod(legacyMod); err != nil {
		t.Fatalf("failed to insert legacy record: %v", err)
	}
	if err := svc.Remove("Legacy"); err != nil {
		t.Fatalf("failed to remove legacy archive record: %v", err)
	}
	if PathExists(legacyDir) {
		t.Error("legacy archive directory should be removed")
	}
}

func TestServiceRemoveRestoresFilesWhenDatabaseDeleteFails(t *testing.T) {
	svc, repo, modsDir, srcDir := setupTestService(t)
	zipPath := filepath.Join(srcDir, "remove_rollback.zip")
	createTestZip(t, zipPath, map[string]string{"one.package": "one", "two.package": "two"})
	installed, err := svc.Add(zipPath, AddOptions{Name: "Keep"})
	if err != nil {
		t.Fatalf("failed to add archive: %v", err)
	}

	svc.repo = failingRepository{Repository: repo, deleteErr: errors.New("delete failed")}
	if err := svc.Remove("Keep"); err == nil {
		t.Fatal("expected database delete failure")
	}
	for name, content := range map[string]string{"one.package": "one", "two.package": "two"} {
		got, err := os.ReadFile(filepath.Join(modsDir, name))
		if err != nil || string(got) != content {
			t.Errorf("failed removal did not restore %s: %q, %v", name, got, err)
		}
	}
	if _, err := repo.GetModByID(installed.ID); err != nil {
		t.Errorf("database record should remain after failed removal: %v", err)
	}
}

func TestServiceRemoveValidatesAllOwnedPathsBeforeChangingFiles(t *testing.T) {
	svc, repo, modsDir, srcDir := setupTestService(t)
	ownedPath := filepath.Join(modsDir, "owned.package")
	outsidePath := filepath.Join(srcDir, "outside.package")
	if err := os.WriteFile(ownedPath, []byte("owned"), 0644); err != nil {
		t.Fatalf("failed to create owned file: %v", err)
	}
	if err := os.WriteFile(outsidePath, []byte("outside"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}
	mod := &db.Mod{
		Name:           "Invalid Paths",
		FileName:       filepath.Base(ownedPath),
		FilePath:       ownedPath,
		InstalledFiles: []string{ownedPath, outsidePath},
		FileHash:       "invalid-paths-hash",
	}
	if err := repo.InsertMod(mod); err != nil {
		t.Fatalf("failed to insert test record: %v", err)
	}

	if err := svc.Remove(mod.Name); err == nil {
		t.Fatal("expected removal of an out-of-Mods path to fail")
	}
	for path, content := range map[string]string{ownedPath: "owned", outsidePath: "outside"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != content {
			t.Errorf("path %s changed after rejected removal: %q, %v", path, got, err)
		}
	}
}

func TestServiceUpdateLegacyArchiveDirectory(t *testing.T) {
	svc, repo, modsDir, srcDir := setupTestService(t)
	legacyDir := filepath.Join(modsDir, "legacy_pack")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy archive directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "old.package"), []byte("legacy"), 0644); err != nil {
		t.Fatalf("failed to create legacy archive file: %v", err)
	}
	legacyMod := &db.Mod{Name: "Legacy", FileName: "legacy_pack", FilePath: legacyDir, FileHash: "legacy-hash"}
	if err := repo.InsertMod(legacyMod); err != nil {
		t.Fatalf("failed to insert legacy record: %v", err)
	}

	newPath := filepath.Join(srcDir, "legacy_v2.package")
	if err := os.WriteFile(newPath, []byte("new-version"), 0644); err != nil {
		t.Fatalf("failed to create new mod file: %v", err)
	}
	if _, err := svc.Update("Legacy", newPath, UpdateOptions{}); err != nil {
		t.Fatalf("failed to update legacy archive record: %v", err)
	}
	if PathExists(legacyDir) || !PathExists(filepath.Join(modsDir, "legacy_v2.package")) {
		t.Error("legacy archive directory was not replaced by the updated mod file")
	}
}

func TestServiceRemove(t *testing.T) {
	svc, _, modsDir, srcDir := setupTestService(t)

	pkgPath := filepath.Join(srcDir, "toBeDeleted.package")
	_ = os.WriteFile(pkgPath, []byte("delete-me"), 0644)

	mod, err := svc.Add(pkgPath, AddOptions{Name: "Deletable"})
	if err != nil {
		t.Fatalf("failed to add mod: %v", err)
	}

	installedPath := filepath.Join(modsDir, "toBeDeleted.package")
	if !PathExists(installedPath) {
		t.Fatalf("mod file does not exist after add")
	}

	if err := svc.Remove("Deletable"); err != nil {
		t.Fatalf("failed to remove mod: %v", err)
	}

	if PathExists(installedPath) {
		t.Errorf("expected file to be deleted from disk")
	}

	_, err = svc.Get(string(rune('0' + int(mod.ID))))
	if !errors.Is(err, db.ErrModNotFound) {
		t.Errorf("expected ErrModNotFound after removal, got: %v", err)
	}
}
