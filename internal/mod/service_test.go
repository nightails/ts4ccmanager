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

	destFolder := filepath.Join(modsDir, "furniture_pack")
	if !PathExists(filepath.Join(destFolder, "sofa.package")) {
		t.Errorf("expected extracted sofa.package in %s", destFolder)
	}
	if !PathExists(filepath.Join(destFolder, "table.package")) {
		t.Errorf("expected extracted table.package in %s", destFolder)
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
