package mod

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ts4ccmanager/internal/db"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported file format: only .package, .ts4script, and .zip are supported")
	ErrDuplicateMod      = errors.New("mod is already installed (duplicate checksum found)")
	errNoSupportedFiles  = errors.New("archive contains no supported files")
)

// AddOptions holds optional metadata when adding a mod.
type AddOptions struct {
	Name    string
	Version string
	Author  string
}

// UpdateOptions holds optional metadata updates.
type UpdateOptions struct {
	Name    string
	Version string
	Author  string
}

// Service provides high-level operations for mod management.
type Service struct {
	repo    db.Repository
	modsDir string
}

// NewService creates a new mod management service.
func NewService(repo db.Repository, modsDir string) *Service {
	return &Service{
		repo:    repo,
		modsDir: modsDir,
	}
}

// Add installs a mod file or archive into the Mods folder and registers it in the database.
func (s *Service) Add(srcPath string, opts AddOptions) (*db.Mod, error) {
	if !PathExists(srcPath) {
		return nil, fmt.Errorf("source mod file does not exist: %s", srcPath)
	}

	if !IsSupportedFormat(srcPath) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, srcPath)
	}

	hash, err := CalculateFileHash(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Check if already registered by hash
	if existing, err := s.repo.GetModByHash(hash); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: %s (ID: %d)", ErrDuplicateMod, existing.Name, existing.ID)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := filepath.Base(srcPath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	modName := opts.Name
	if modName == "" {
		modName = nameWithoutExt
	}

	stageDir, stagedFiles, err := s.stageFiles(srcPath, ext)
	if err != nil {
		if ext == ".zip" && errors.Is(err, errNoSupportedFiles) {
			return nil, fmt.Errorf("zip archive contains no supported files: %s", srcPath)
		}
		return nil, fmt.Errorf("failed to stage mod files: %w", err)
	}
	defer func() { _ = RemovePath(stageDir) }()

	installedFiles, _, retainStage, err := s.installStagedFiles(stageDir, stagedFiles, nil)
	if err != nil {
		if retainStage {
			stageDir = ""
		}
		return nil, fmt.Errorf("failed to install mod files: %w", err)
	}
	targetPath := installedFiles[0]
	targetFileName := filepath.Base(targetPath)
	trackedFiles := []string(nil)
	if ext == ".zip" {
		trackedFiles = installedFiles
	}

	mod := &db.Mod{
		Name:           modName,
		FileName:       targetFileName,
		FilePath:       targetPath,
		InstalledFiles: trackedFiles,
		FileHash:       hash,
		Version:        opts.Version,
		Author:         opts.Author,
		InstalledAt:    time.Now().UTC(),
		Status:         "active",
	}

	if err := s.repo.InsertMod(mod); err != nil {
		if rollbackErr := rollbackInstalledFiles(installedFiles, nil); rollbackErr != nil {
			stageDir = ""
			return nil, fmt.Errorf("failed to register mod in database: %v; failed to roll back installed files: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("failed to register mod in database: %w", err)
	}

	return mod, nil
}

// List returns all registered mods.
func (s *Service) List() ([]*db.Mod, error) {
	return s.repo.ListMods()
}

// Get finds a mod by ID or name.
func (s *Service) Get(identifier string) (*db.Mod, error) {
	return s.repo.FindModByIDOrName(identifier)
}

// Remove deletes a mod from both disk and the database.
func (s *Service) Remove(identifier string) error {
	mod, err := s.repo.FindModByIDOrName(identifier)
	if err != nil {
		return err
	}

	stageDir, err := s.newStageDir()
	if err != nil {
		return fmt.Errorf("failed to prepare mod removal: %w", err)
	}
	backups, retainStage, err := backupManagedFiles(s.modsDir, stageDir, managedFilePaths(mod))
	if err != nil {
		if retainStage {
			stageDir = ""
		}
		_ = RemovePath(stageDir)
		return fmt.Errorf("failed to delete mod file/directory from disk: %w", err)
	}

	if err := s.repo.DeleteMod(mod.ID); err != nil {
		if rollbackErr := restoreBackups(backups); rollbackErr != nil {
			stageDir = ""
			return fmt.Errorf("failed to remove mod record from database: %v; failed to restore mod files: %w", err, rollbackErr)
		}
		_ = RemovePath(stageDir)
		return fmt.Errorf("failed to remove mod record from database: %w", err)
	}

	_ = RemovePath(stageDir)
	return nil
}

// Update replaces an existing mod's file and updates its metadata in the database.
func (s *Service) Update(identifier string, newFilePath string, opts UpdateOptions) (*db.Mod, error) {
	mod, err := s.repo.FindModByIDOrName(identifier)
	if err != nil {
		return nil, err
	}

	if !PathExists(newFilePath) {
		return nil, fmt.Errorf("new mod file does not exist: %s", newFilePath)
	}

	if !IsSupportedFormat(newFilePath) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, newFilePath)
	}

	newHash, err := CalculateFileHash(newFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum of new file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(newFilePath))

	stageDir, stagedFiles, err := s.stageFiles(newFilePath, ext)
	if err != nil {
		if ext == ".zip" && errors.Is(err, errNoSupportedFiles) {
			return nil, fmt.Errorf("zip archive contains no supported files: %s", newFilePath)
		}
		return nil, fmt.Errorf("failed to stage updated mod files: %w", err)
	}
	retainStage := false
	defer func() {
		if !retainStage {
			_ = RemovePath(stageDir)
		}
	}()

	installedFiles, backups, shouldRetain, err := s.installStagedFiles(stageDir, stagedFiles, managedFilePaths(mod))
	if err != nil {
		retainStage = shouldRetain
		return nil, fmt.Errorf("failed to install updated mod files: %w", err)
	}

	original := *mod
	original.InstalledFiles = append([]string(nil), mod.InstalledFiles...)
	mod.FileName = filepath.Base(installedFiles[0])
	mod.FilePath = installedFiles[0]
	mod.InstalledFiles = nil
	if ext == ".zip" {
		mod.InstalledFiles = installedFiles
	}
	mod.FileHash = newHash
	if opts.Name != "" {
		mod.Name = opts.Name
	}
	if opts.Version != "" {
		mod.Version = opts.Version
	}
	if opts.Author != "" {
		mod.Author = opts.Author
	}

	if err := s.repo.UpdateMod(mod); err != nil {
		*mod = original
		if rollbackErr := rollbackInstalledFiles(installedFiles, backups); rollbackErr != nil {
			retainStage = true
			return nil, fmt.Errorf("failed to update mod record in database: %v; failed to restore files: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("failed to update mod record in database: %w", err)
	}

	return mod, nil
}

func (s *Service) newStageDir() (string, error) {
	if err := os.MkdirAll(s.modsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create mods directory: %w", err)
	}
	stageDir, err := os.MkdirTemp(s.modsDir, ".ts4ccmanager-")
	if err != nil {
		return "", fmt.Errorf("failed to create staging directory: %w", err)
	}
	return stageDir, nil
}

func (s *Service) stageFiles(sourcePath, extension string) (string, []string, error) {
	stageDir, err := s.newStageDir()
	if err != nil {
		return "", nil, err
	}
	filesDir := filepath.Join(stageDir, "files")
	var stagedFiles []string
	if extension == ".zip" {
		stagedFiles, err = ExtractZip(sourcePath, filesDir)
		if err == nil && len(stagedFiles) == 0 {
			err = errNoSupportedFiles
		}
	} else {
		if err := os.MkdirAll(filesDir, 0755); err != nil {
			_ = RemovePath(stageDir)
			return "", nil, fmt.Errorf("failed to create staging directory: %w", err)
		}
		stagedPath := filepath.Join(filesDir, filepath.Base(sourcePath))
		if err := CopyFile(sourcePath, stagedPath); err != nil {
			_ = RemovePath(stageDir)
			return "", nil, err
		}
		stagedFiles = []string{stagedPath}
	}
	if err != nil {
		_ = RemovePath(stageDir)
		return "", nil, err
	}
	return stageDir, stagedFiles, nil
}

type fileBackup struct {
	original string
	backup   string
}

func (s *Service) installStagedFiles(stageDir string, stagedFiles, oldPaths []string) ([]string, []fileBackup, bool, error) {
	ownedPaths := make(map[string]bool, len(oldPaths))
	for _, oldPath := range oldPaths {
		key, err := s.managedPathKey(oldPath)
		if err != nil {
			return nil, nil, false, err
		}
		ownedPaths[key] = true
	}

	targets := make([]string, len(stagedFiles))
	targetNames := make(map[string]bool, len(stagedFiles))
	for i, stagedPath := range stagedFiles {
		name := filepath.Base(stagedPath)
		if name == "." || name == string(os.PathSeparator) {
			return nil, nil, false, fmt.Errorf("invalid staged filename: %s", stagedPath)
		}
		key := strings.ToLower(name)
		if targetNames[key] {
			return nil, nil, false, fmt.Errorf("duplicate installed filename: %s", name)
		}
		targetNames[key] = true
		targets[i] = filepath.Join(s.modsDir, name)
		_, err := os.Lstat(targets[i])
		if err == nil {
			managedKey, keyErr := s.managedPathKey(targets[i])
			if keyErr != nil {
				return nil, nil, false, keyErr
			}
			if !ownedPaths[managedKey] {
				return nil, nil, false, fmt.Errorf("destination already exists: %s", targets[i])
			}
		} else if !os.IsNotExist(err) {
			return nil, nil, false, fmt.Errorf("failed to inspect destination %s: %w", targets[i], err)
		}
	}

	backups, retainStage, err := backupManagedFiles(s.modsDir, stageDir, oldPaths)
	if err != nil {
		return nil, nil, retainStage, err
	}
	var installed []string
	for i, stagedPath := range stagedFiles {
		if err := copyFileExclusive(stagedPath, targets[i]); err != nil {
			rollbackErr := rollbackInstalledFiles(installed, backups)
			if rollbackErr != nil {
				return nil, nil, true, fmt.Errorf("failed to copy %s: %v; rollback failed: %w", targets[i], err, rollbackErr)
			}
			return nil, nil, false, fmt.Errorf("failed to copy %s: %w", targets[i], err)
		}
		installed = append(installed, targets[i])
	}
	return installed, backups, false, nil
}

func backupManagedFiles(modsDir, stageDir string, paths []string) ([]fileBackup, bool, error) {
	backupDir := filepath.Join(stageDir, "backup")
	uniquePaths := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, original := range paths {
		absPath, err := filepath.Abs(original)
		if err != nil {
			return nil, false, err
		}
		key := strings.ToLower(filepath.Clean(absPath))
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := ensureWithinModsDir(modsDir, original); err != nil {
			return nil, false, err
		}
		uniquePaths = append(uniquePaths, original)
	}

	backups := make([]fileBackup, 0, len(uniquePaths))
	for _, original := range uniquePaths {
		if _, err := os.Lstat(original); os.IsNotExist(err) {
			continue
		} else if err != nil {
			rollbackErr := restoreBackups(backups)
			if rollbackErr != nil {
				return backups, true, fmt.Errorf("failed to inspect managed path %s: %v; rollback failed: %w", original, err, rollbackErr)
			}
			return nil, false, fmt.Errorf("failed to inspect managed path %s: %w", original, err)
		}
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			rollbackErr := restoreBackups(backups)
			if rollbackErr != nil {
				return backups, true, fmt.Errorf("failed to create backup directory: %v; rollback failed: %w", err, rollbackErr)
			}
			return nil, false, fmt.Errorf("failed to create backup directory: %w", err)
		}
		backupPath := filepath.Join(backupDir, fmt.Sprintf("%d", len(backups)))
		if err := os.Rename(original, backupPath); err != nil {
			rollbackErr := restoreBackups(backups)
			if rollbackErr != nil {
				return backups, true, fmt.Errorf("failed to back up %s: %v; rollback failed: %w", original, err, rollbackErr)
			}
			return nil, false, fmt.Errorf("failed to back up %s: %w", original, err)
		}
		backups = append(backups, fileBackup{original: original, backup: backupPath})
	}
	return backups, false, nil
}

func rollbackInstalledFiles(installed []string, backups []fileBackup) error {
	for i := len(installed) - 1; i >= 0; i-- {
		if err := os.Remove(installed[i]); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", installed[i], err)
		}
	}
	return restoreBackups(backups)
}

func restoreBackups(backups []fileBackup) error {
	for i := len(backups) - 1; i >= 0; i-- {
		backup := backups[i]
		if err := os.MkdirAll(filepath.Dir(backup.original), 0755); err != nil {
			return fmt.Errorf("failed to recreate parent for %s: %w", backup.original, err)
		}
		if err := os.Rename(backup.backup, backup.original); err != nil {
			return fmt.Errorf("failed to restore %s: %w", backup.original, err)
		}
	}
	return nil
}

func copyFileExclusive(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open staged file: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("failed to inspect staged file: %w", err)
	}
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("failed to copy file contents: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("failed to close destination file: %w", closeErr)
	}
	return nil
}

func managedFilePaths(mod *db.Mod) []string {
	if len(mod.InstalledFiles) > 0 {
		return append([]string(nil), mod.InstalledFiles...)
	}
	if mod.FilePath == "" {
		return nil
	}
	return []string{mod.FilePath}
}

func (s *Service) managedPathKey(filePath string) (string, error) {
	if err := ensureWithinModsDir(s.modsDir, filePath); err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	return strings.ToLower(filepath.Clean(absPath)), nil
}

func ensureWithinModsDir(modsDir, filePath string) error {
	root, err := filepath.Abs(modsDir)
	if err != nil {
		return err
	}
	path, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("managed path is outside Mods directory: %s", filePath)
	}
	return nil
}
