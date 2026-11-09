package mod

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestExtractZipOnlyCopiesSupportedFiles(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "mods.zip")
	destDir := filepath.Join(tempDir, "mods")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}
	zipWriter := zip.NewWriter(zipFile)
	files := map[string]string{
		"hair.package":          "package-content",
		"scripts/mod.ts4script": "script-content",
		"readme.txt":            "readme-content",
		"nested/archive.zip":    "nested-zip-content",
		"scripts/notes.TXT":     "notes-content",
		"scripts/UPPER.PACKAGE": "uppercase-package-content",
	}
	for name, content := range files {
		entry, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("failed to add %q to zip: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write %q to zip: %v", name, err)
		}
	}
	if _, err := zipWriter.Create("empty/"); err != nil {
		t.Fatalf("failed to add directory to zip: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	extracted, err := ExtractZip(zipPath, destDir)
	if err != nil {
		t.Fatalf("unexpected extraction error: %v", err)
	}

	wantPaths := []string{
		filepath.Join(destDir, "hair.package"),
		filepath.Join(destDir, "mod.ts4script"),
		filepath.Join(destDir, "UPPER.PACKAGE"),
	}
	sort.Strings(extracted)
	sort.Strings(wantPaths)
	if !reflect.DeepEqual(extracted, wantPaths) {
		t.Errorf("unexpected extracted paths: got %v, want %v", extracted, wantPaths)
	}
	for _, name := range []string{"readme.txt", "nested/archive.zip", "scripts/notes.TXT", "scripts/mod.ts4script", "scripts/UPPER.PACKAGE"} {
		if _, err := os.Stat(filepath.Join(destDir, filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Errorf("archive path %q should not be preserved in the flattened output", name)
		}
	}
	if _, err := os.Stat(filepath.Join(destDir, "empty")); !os.IsNotExist(err) {
		t.Errorf("empty archive directory should not be created")
	}
}

func TestExtractZipRejectsDuplicateBasenames(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "mods.zip")
	createTestZip(t, zipPath, map[string]string{
		"one/shared.package": "first",
		"two/shared.package": "second",
	})

	if _, err := ExtractZip(zipPath, filepath.Join(tempDir, "mods")); err == nil {
		t.Fatal("expected duplicate flattened filenames to be rejected")
	}
}

func TestExtractZipRejectsTraversalPath(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "mods.zip")
	createTestZip(t, zipPath, map[string]string{"../outside.package": "content"})

	if _, err := ExtractZip(zipPath, filepath.Join(tempDir, "mods")); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "outside.package")); !os.IsNotExist(err) {
		t.Errorf("traversal target should not be created")
	}
}
