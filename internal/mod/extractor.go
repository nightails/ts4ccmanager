package mod

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var supportedExtensions = map[string]bool{
	".package":   true,
	".ts4script": true,
	".zip":       true,
}

// IsSupportedFormat returns true if the file extension is supported by ts4ccmanager.
func IsSupportedFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return supportedExtensions[ext]
}

// ExtractZip extracts a zip archive into destDir while preventing zip slip vulnerabilities.
// Returns a list of extracted file paths.
func ExtractZip(zipPath string, destDir string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	seenNames := make(map[string]bool)
	cleanDestDir := filepath.Clean(destDir)

	for _, f := range r.File {
		entryPath := filepath.Clean(filepath.FromSlash(f.Name))
		if filepath.IsAbs(entryPath) || filepath.VolumeName(entryPath) != "" || entryPath == ".." || strings.HasPrefix(entryPath, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}
		targetPath := filepath.Join(cleanDestDir, entryPath)
		cleanTarget := filepath.Clean(targetPath)

		// Security check: zip-slip prevention
		if !strings.HasPrefix(cleanTarget, cleanDestDir+string(os.PathSeparator)) && cleanTarget != cleanDestDir {
			return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".package" && ext != ".ts4script" {
			continue
		}

		fileName := filepath.Base(entryPath)
		nameKey := strings.ToLower(fileName)
		if seenNames[nameKey] {
			return nil, fmt.Errorf("duplicate supported filename in zip: %s", fileName)
		}
		seenNames[nameKey] = true
		cleanTarget = filepath.Join(cleanDestDir, fileName)

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s in zip: %w", f.Name, err)
		}

		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_EXCL, f.Mode())
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("failed to create output file %s: %w", cleanTarget, err)
		}

		_, err = io.Copy(outFile, rc)
		_ = outFile.Close()
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to write output file %s: %w", cleanTarget, err)
		}

		extractedFiles = append(extractedFiles, cleanTarget)
	}

	return extractedFiles, nil
}
