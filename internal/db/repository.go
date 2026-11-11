package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrModNotFound      = errors.New("mod not found")
	ErrModAlreadyExists = errors.New("mod already exists with this path or checksum")
)

// Repository defines operations for querying and persisting mod metadata.
type Repository interface {
	InsertMod(mod *Mod) error
	GetModByID(id int64) (*Mod, error)
	GetModByName(name string) (*Mod, error)
	GetModByPath(filePath string) (*Mod, error)
	GetModByHash(fileHash string) (*Mod, error)
	FindModByIDOrName(identifier string) (*Mod, error)
	ListMods() ([]*Mod, error)
	UpdateMod(mod *Mod) error
	DeleteMod(id int64) error
	Close() error
}

// SQLiteRepository implements Repository backed by SQLite.
type SQLiteRepository struct {
	db      *sql.DB
	queries *Queries
}

// NewRepository creates a new SQLiteRepository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db, queries: New(db)}
}

// Close closes the underlying database connection.
func (r *SQLiteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// InsertMod inserts a new mod record into the database.
func (r *SQLiteRepository) InsertMod(mod *Mod) error {
	if mod.InstalledAt.IsZero() {
		mod.InstalledAt = time.Now().UTC()
	}
	if mod.Status == "" {
		mod.Status = "active"
	}

	installedFiles, err := marshalInstalledFiles(mod.InstalledFiles)
	if err != nil {
		return fmt.Errorf("failed to encode installed files: %w", err)
	}
	res, err := r.queries.InsertMod(context.Background(), InsertModParams{
		Name:           mod.Name,
		FileName:       mod.FileName,
		FilePath:       mod.FilePath,
		InstalledFiles: installedFiles,
		FileHash:       mod.FileHash,
		Version:        sql.NullString{String: mod.Version, Valid: true},
		Author:         sql.NullString{String: mod.Author, Valid: true},
		InstalledAt:    sql.NullTime{Time: mod.InstalledAt, Valid: true},
		Status:         sql.NullString{String: mod.Status, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to insert mod: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to retrieve last insert id: %w", err)
	}
	mod.ID = id
	return nil
}

// GetModByID retrieves a mod by its primary ID.
func (r *SQLiteRepository) GetModByID(id int64) (*Mod, error) {
	record, err := r.queries.GetModByID(context.Background(), id)
	return mapModRecord(record, err)
}

// GetModByName retrieves a mod by its name.
func (r *SQLiteRepository) GetModByName(name string) (*Mod, error) {
	record, err := r.queries.GetModByName(context.Background(), name)
	return mapModRecord(record, err)
}

// GetModByPath retrieves a mod by its file path.
func (r *SQLiteRepository) GetModByPath(filePath string) (*Mod, error) {
	record, err := r.queries.GetModByPath(context.Background(), filePath)
	return mapModRecord(record, err)
}

// GetModByHash retrieves a mod by its SHA256 checksum.
func (r *SQLiteRepository) GetModByHash(fileHash string) (*Mod, error) {
	record, err := r.queries.GetModByHash(context.Background(), fileHash)
	return mapModRecord(record, err)
}

// FindModByIDOrName attempts to find a mod by numeric ID, falling back to name lookup.
func (r *SQLiteRepository) FindModByIDOrName(identifier string) (*Mod, error) {
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		m, err := r.GetModByID(id)
		if err == nil {
			return m, nil
		}
		if !errors.Is(err, ErrModNotFound) {
			return nil, err
		}
	}
	return r.GetModByName(identifier)
}

// ListMods returns all installed mods sorted by ID.
func (r *SQLiteRepository) ListMods() ([]*Mod, error) {
	records, err := r.queries.ListMods(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to query mods: %w", err)
	}

	var mods []*Mod
	for _, record := range records {
		m, err := mapModRecord(record, nil)
		if err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// UpdateMod updates mod metadata for an existing record.
func (r *SQLiteRepository) UpdateMod(mod *Mod) error {
	installedFiles, err := marshalInstalledFiles(mod.InstalledFiles)
	if err != nil {
		return fmt.Errorf("failed to encode installed files: %w", err)
	}
	res, err := r.queries.UpdateMod(context.Background(), UpdateModParams{
		Name:           mod.Name,
		FileName:       mod.FileName,
		FilePath:       mod.FilePath,
		InstalledFiles: installedFiles,
		FileHash:       mod.FileHash,
		Version:        sql.NullString{String: mod.Version, Valid: true},
		Author:         sql.NullString{String: mod.Author, Valid: true},
		Status:         sql.NullString{String: mod.Status, Valid: true},
		ID:             mod.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update mod: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrModNotFound
	}

	return nil
}

// DeleteMod deletes a mod record by ID.
func (r *SQLiteRepository) DeleteMod(id int64) error {
	res, err := r.queries.DeleteMod(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to delete mod: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrModNotFound
	}

	return nil
}

func mapModRecord(record ModRecord, err error) (*Mod, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrModNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan mod: %w", err)
	}
	var installedFiles []string
	if err := json.Unmarshal([]byte(record.InstalledFiles), &installedFiles); err != nil {
		return nil, fmt.Errorf("failed to decode installed files: %w", err)
	}
	mod := &Mod{
		ID:             record.ID,
		Name:           record.Name,
		FileName:       record.FileName,
		FilePath:       record.FilePath,
		InstalledFiles: installedFiles,
		FileHash:       record.FileHash,
		Version:        record.Version.String,
		Author:         record.Author.String,
		Status:         record.Status.String,
	}
	if record.InstalledAt.Valid {
		mod.InstalledAt = record.InstalledAt.Time
	}
	return mod, nil
}

func marshalInstalledFiles(files []string) (string, error) {
	if files == nil {
		files = []string{}
	}
	encoded, err := json.Marshal(files)
	return string(encoded), err
}
