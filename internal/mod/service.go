package mod

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ts4ccmanager/internal/db"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported file format: only .package, .ts4script, and .zip are supported")
	ErrDuplicateMod      = errors.New("mod is already installed (duplicate checksum found)")
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

	var targetPath string
	var targetFileName string

	if ext == ".zip" {
		targetFolderName := nameWithoutExt
		targetPath = filepath.Join(s.modsDir, targetFolderName)
		targetFileName = targetFolderName

		extracted, err := ExtractZip(srcPath, targetPath)
		if err != nil {
			_ = RemovePath(targetPath)
			return nil, fmt.Errorf("failed to extract zip archive: %w", err)
		}
		if len(extracted) == 0 {
			_ = RemovePath(targetPath)
			return nil, fmt.Errorf("zip archive contains no supported files: %s", srcPath)
		}
	} else {
		targetFileName = baseName
		targetPath = filepath.Join(s.modsDir, targetFileName)

		if err := CopyFile(srcPath, targetPath); err != nil {
			return nil, fmt.Errorf("failed to copy mod to mods directory: %w", err)
		}
	}

	mod := &db.Mod{
		Name:        modName,
		FileName:    targetFileName,
		FilePath:    targetPath,
		FileHash:    hash,
		Version:     opts.Version,
		Author:      opts.Author,
		InstalledAt: time.Now().UTC(),
		Status:      "active",
	}

	if err := s.repo.InsertMod(mod); err != nil {
		// Roll back filesystem change
		_ = RemovePath(targetPath)
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

	// Delete from filesystem
	if err := RemovePath(mod.FilePath); err != nil {
		return fmt.Errorf("failed to delete mod file/directory from disk: %w", err)
	}

	// Delete from database
	if err := s.repo.DeleteMod(mod.ID); err != nil {
		return fmt.Errorf("failed to remove mod record from database: %w", err)
	}

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
	baseName := filepath.Base(newFilePath)
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Clean up old file/directory
	_ = RemovePath(mod.FilePath)

	var targetPath string
	var targetFileName string

	if ext == ".zip" {
		targetFolderName := nameWithoutExt
		targetPath = filepath.Join(s.modsDir, targetFolderName)
		targetFileName = targetFolderName

		_, err := ExtractZip(newFilePath, targetPath)
		if err != nil {
			_ = RemovePath(targetPath)
			return nil, fmt.Errorf("failed to extract updated zip archive: %w", err)
		}
	} else {
		targetFileName = baseName
		targetPath = filepath.Join(s.modsDir, targetFileName)

		if err := CopyFile(newFilePath, targetPath); err != nil {
			return nil, fmt.Errorf("failed to copy updated mod file: %w", err)
		}
	}

	mod.FileName = targetFileName
	mod.FilePath = targetPath
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
		return nil, fmt.Errorf("failed to update mod record in database: %w", err)
	}

	return mod, nil
}
