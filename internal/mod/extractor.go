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
	cleanDestDir := filepath.Clean(destDir)

	for _, f := range r.File {
		targetPath := filepath.Join(cleanDestDir, f.Name)
		cleanTarget := filepath.Clean(targetPath)

		// Security check: zip-slip prevention
		if !strings.HasPrefix(cleanTarget, cleanDestDir+string(os.PathSeparator)) && cleanTarget != cleanDestDir {
			return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", cleanTarget, err)
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s: %w", cleanTarget, err)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s in zip: %w", f.Name, err)
		}

		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
