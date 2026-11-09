package db

import (
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
	db *sql.DB
}

// NewRepository creates a new SQLiteRepository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
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
	query := `
		INSERT INTO mods (name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.Exec(query, mod.Name, mod.FileName, mod.FilePath, installedFiles, mod.FileHash, mod.Version, mod.Author, mod.InstalledAt, mod.Status)
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
	query := `
		SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
		FROM mods
		WHERE id = ?
	`
	row := r.db.QueryRow(query, id)
	return scanMod(row)
}

// GetModByName retrieves a mod by its name.
func (r *SQLiteRepository) GetModByName(name string) (*Mod, error) {
	query := `
		SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
		FROM mods
		WHERE name = ?
		LIMIT 1
	`
	row := r.db.QueryRow(query, name)
	return scanMod(row)
}

// GetModByPath retrieves a mod by its file path.
func (r *SQLiteRepository) GetModByPath(filePath string) (*Mod, error) {
	query := `
		SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
		FROM mods
		WHERE file_path = ?
		LIMIT 1
	`
	row := r.db.QueryRow(query, filePath)
	return scanMod(row)
}

// GetModByHash retrieves a mod by its SHA256 checksum.
func (r *SQLiteRepository) GetModByHash(fileHash string) (*Mod, error) {
	query := `
		SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
		FROM mods
		WHERE file_hash = ?
		LIMIT 1
	`
	row := r.db.QueryRow(query, fileHash)
	return scanMod(row)
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
	query := `
		SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
		FROM mods
		ORDER BY id ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query mods: %w", err)
	}
	defer rows.Close()

	var mods []*Mod
	for rows.Next() {
		m, err := scanMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during mods row iteration: %w", err)
	}

	return mods, nil
}

// UpdateMod updates mod metadata for an existing record.
func (r *SQLiteRepository) UpdateMod(mod *Mod) error {
	installedFiles, err := marshalInstalledFiles(mod.InstalledFiles)
	if err != nil {
		return fmt.Errorf("failed to encode installed files: %w", err)
	}
	query := `
		UPDATE mods
		SET name = ?, file_name = ?, file_path = ?, installed_files = ?, file_hash = ?, version = ?, author = ?, status = ?
		WHERE id = ?
	`
	res, err := r.db.Exec(query, mod.Name, mod.FileName, mod.FilePath, installedFiles, mod.FileHash, mod.Version, mod.Author, mod.Status, mod.ID)
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
	query := `DELETE FROM mods WHERE id = ?`
	res, err := r.db.Exec(query, id)
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMod(scanner rowScanner) (*Mod, error) {
	var m Mod
	var installedFiles string
	if err := scanner.Scan(&m.ID, &m.Name, &m.FileName, &m.FilePath, &installedFiles, &m.FileHash, &m.Version, &m.Author, &m.InstalledAt, &m.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrModNotFound
		}
		return nil, fmt.Errorf("failed to scan mod: %w", err)
	}
	if err := json.Unmarshal([]byte(installedFiles), &m.InstalledFiles); err != nil {
		return nil, fmt.Errorf("failed to decode installed files: %w", err)
	}
	return &m, nil
}

func marshalInstalledFiles(files []string) (string, error) {
	if files == nil {
		files = []string{}
	}
	encoded, err := json.Marshal(files)
	return string(encoded), err
}
